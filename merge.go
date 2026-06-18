package dom

import (
	"errors"
	"fmt"

	yaml "go.yaml.in/yaml/v4"
)

type SequenceStrategy int

const (
	SequenceReplace SequenceStrategy = iota
	SequenceAppend
	SequenceByIndex
)

type KeyFilter int

const (
	KeyAll KeyFilter = iota
	KeyOnlyExisting
	KeyOnlyNew
)

type NullStrategy int

const (
	NullOverwrite NullStrategy = iota
	NullIgnore
	NullDelete
)

type mergeConfig struct {
	sequence SequenceStrategy
	keys     KeyFilter
	clobber  bool
	null     NullStrategy
	shared   bool
}

func defaultMergeConfig() mergeConfig {
	return mergeConfig{sequence: SequenceReplace, keys: KeyAll, null: NullOverwrite}
}

type MergeOption func(*mergeConfig)

func WithSequenceMerge(s SequenceStrategy) MergeOption {
	return func(c *mergeConfig) { c.sequence = s }
}
func WithAppendSequences() MergeOption         { return WithSequenceMerge(SequenceAppend) }
func WithKeyFilter(f KeyFilter) MergeOption    { return func(c *mergeConfig) { c.keys = f } }
func WithOnlyExistingKeys() MergeOption        { return WithKeyFilter(KeyOnlyExisting) }
func WithOnlyNewKeys() MergeOption             { return WithKeyFilter(KeyOnlyNew) }
func WithNullMerge(s NullStrategy) MergeOption { return func(c *mergeConfig) { c.null = s } }

func firstBool(def bool, b []bool) bool {
	if len(b) > 0 {
		return b[0]
	}
	return def
}

func WithClobberTags(enable ...bool) MergeOption {
	return func(c *mergeConfig) { c.clobber = firstBool(true, enable) }
}

func WithSharedSource(enable ...bool) MergeOption {
	return func(c *mergeConfig) { c.shared = firstBool(true, enable) }
}

// Merge deep-merges src into dst in place. dst is mutated; src is not unless
// WithSharedSource is set, in which case src subtrees may be grafted directly.
func Merge(dst, src *yaml.Node, opts ...MergeOption) error {
	if dst == nil {
		return errors.New("dom: Merge dst is nil")
	}
	if src == nil {
		return errors.New("dom: Merge src is nil")
	}
	cfg := defaultMergeConfig()
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	if err := validateMergeConfig(cfg); err != nil {
		return err
	}
	return mergeNode(unwrapDoc(dst), unwrapDoc(src), cfg)
}

func validateMergeConfig(cfg mergeConfig) error {
	if cfg.sequence < SequenceReplace || cfg.sequence > SequenceByIndex {
		return fmt.Errorf("dom: unknown sequence merge strategy %d", cfg.sequence)
	}
	if cfg.keys < KeyAll || cfg.keys > KeyOnlyNew {
		return fmt.Errorf("dom: unknown key filter %d", cfg.keys)
	}
	if cfg.null < NullOverwrite || cfg.null > NullDelete {
		return fmt.Errorf("dom: unknown null merge strategy %d", cfg.null)
	}
	return nil
}

func unwrapDoc(n *yaml.Node) *yaml.Node {
	if n != nil && n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		return n.Content[0]
	}
	return n
}

func mergeNode(dst, src *yaml.Node, cfg mergeConfig) error {
	if dst == nil || src == nil {
		return errors.New("dom: cannot merge nil node")
	}
	if isNull(src) {
		switch cfg.null {
		case NullIgnore, NullDelete:
			return nil
		case NullOverwrite:
			copyNodeInto(dst, src, cfg)
			dst.Tag = src.Tag
			return nil
		}
	}
	if dst.Kind == yaml.MappingNode && src.Kind == yaml.MappingNode {
		if cfg.clobber {
			dst.Tag = src.Tag
		}
		return mergeMapping(dst, src, cfg)
	}
	if dst.Kind == yaml.SequenceNode && src.Kind == yaml.SequenceNode {
		if cfg.clobber {
			dst.Tag = src.Tag
		}
		return mergeSequence(dst, src, cfg)
	}
	copyNodeInto(dst, src, cfg)
	return nil
}

func mergeMapping(dst, src *yaml.Node, cfg mergeConfig) error {
	for i := 0; i+1 < len(src.Content); i += 2 {
		srcKey, srcVal := src.Content[i], src.Content[i+1]
		dstIdx := findMappingKey(dst, srcKey)
		if dstIdx >= 0 {
			if cfg.keys == KeyOnlyNew {
				continue
			}
			if isNull(srcVal) && cfg.null == NullDelete {
				dst.Content = append(dst.Content[:dstIdx], dst.Content[dstIdx+2:]...)
				continue
			}
			if err := mergeNode(dst.Content[dstIdx+1], srcVal, cfg); err != nil {
				return err
			}
			continue
		}
		if cfg.keys == KeyOnlyExisting || (isNull(srcVal) && cfg.null == NullDelete) {
			continue
		}
		dst.Content = append(dst.Content, landNode(srcKey, cfg), landNode(srcVal, cfg))
	}
	return nil
}

func mergeSequence(dst, src *yaml.Node, cfg mergeConfig) error {
	switch cfg.sequence {
	case SequenceReplace:
		dst.Content = cloneNodeSlice(src.Content, cfg)
	case SequenceAppend:
		dst.Content = append(dst.Content, cloneNodeSlice(src.Content, cfg)...)
	case SequenceByIndex:
		n := minInt(len(dst.Content), len(src.Content))
		for i := 0; i < n; i++ {
			if err := mergeNode(dst.Content[i], src.Content[i], cfg); err != nil {
				return err
			}
		}
		dst.Content = append(dst.Content, cloneNodeSlice(src.Content[n:], cfg)...)
	default:
		return fmt.Errorf("dom: unknown sequence merge strategy %d", cfg.sequence)
	}
	return nil
}

func findMappingKey(mapping, key *yaml.Node) int {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mappingKeysEqual(mapping.Content[i], key) {
			return i
		}
	}
	return -1
}

func mappingKeysEqual(a, b *yaml.Node) bool {
	return mappingKeysEqualSeen(a, b, map[[2]*yaml.Node]bool{})
}

func mappingKeysEqualSeen(a, b *yaml.Node, seen map[[2]*yaml.Node]bool) bool {
	if Equal(a, b) {
		return true
	}
	if a == nil || b == nil {
		return a == b
	}
	key := [2]*yaml.Node{a, b}
	if seen[key] {
		return true
	}
	seen[key] = true
	if a.Kind != b.Kind || a.Value != b.Value || len(a.Content) != len(b.Content) {
		return false
	}
	if a.Tag != b.Tag && a.Tag != "" && b.Tag != "" {
		return false
	}
	for i := range a.Content {
		if !mappingKeysEqualSeen(a.Content[i], b.Content[i], seen) {
			return false
		}
	}
	return mappingKeysEqualSeen(a.Alias, b.Alias, seen)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isNull(n *yaml.Node) bool {
	return n != nil && n.Tag == "!!null"
}

func landNode(n *yaml.Node, cfg mergeConfig) *yaml.Node {
	if cfg.shared {
		return n
	}
	return Clone(n)
}

func cloneNodeSlice(nodes []*yaml.Node, cfg mergeConfig) []*yaml.Node {
	if nodes == nil {
		return nil
	}
	out := make([]*yaml.Node, len(nodes))
	for i, n := range nodes {
		out[i] = landNode(n, cfg)
	}
	return out
}

func copyNodeInto(dst, src *yaml.Node, cfg mergeConfig) {
	tag := dst.Tag
	style := dst.Style
	headComment := dst.HeadComment
	lineComment := dst.LineComment
	footComment := dst.FootComment
	line := dst.Line
	column := dst.Column

	replacement := landNode(src, cfg)
	*dst = *replacement
	if !cfg.clobber {
		dst.Tag = tag
	}
	dst.Style = style
	dst.HeadComment = headComment
	dst.LineComment = lineComment
	dst.FootComment = footComment
	dst.Line = line
	dst.Column = column
}
