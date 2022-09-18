package core_test

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v3"
)

func TestYamlNode(t *testing.T) {
	config := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{
						Kind:        yaml.ScalarNode,
						Value:       "key1",
						HeadComment: "# This section is for key1",
						LineComment: "# TODO Complete",
					},
					{
						Kind:  yaml.ScalarNode,
						Style: yaml.DoubleQuotedStyle,
						Value: "value1",
					},

					{
						Kind:  yaml.ScalarNode,
						Value: "key2",
					},
					{
						Kind:  yaml.ScalarNode,
						Style: yaml.DoubleQuotedStyle,
						Value: "value2",
					},
				},
			},
		},
	}

	bytes, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Unable to marshall: %v", err)
	}

	t.Log("\n---\n" + string(bytes) + "---")
}

func TestYamlNodeWithObject(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	me := Person{
		Name: "Julien",
		Age:  36,
	}
	var meNode yaml.Node
	data, err := yaml.Marshal(&me)
	if err != nil {
		t.Fatal(err)
	}
	err = yaml.Unmarshal(data, &meNode)
	if err != nil {
		t.Fatal(err)
	}

	config := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{
						Kind:        yaml.ScalarNode,
						Value:       "creator",
						LineComment: "# Who?",
					},
					meNode.Content[0],
				},
			},
		},
	}

	bytes, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Unable to marshall: %v", err)
	}

	t.Log("\n---\n" + string(bytes) + "---")
}

func TestDump(t *testing.T) {
	t.Skip()
	doc := `
creator:
  name: Julien
  hobbies:
    - name: Running
      frequency: 3/weeks
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(doc), &node)
	if err != nil {
		t.Fatal(err)
	}
	spew.Dump(node)
	t.Fail()

	// (yaml.Node) {
	// 	Kind: DocumentNode,
	// 	Style: "",
	// 	Tag: (string) "",
	// 	Value: (string) "",
	// 	Content: ([]*yaml.Node) (len=1 cap=1) {
	// 	 (*yaml.Node)(0xc0000b8be0)({
	// 	  Kind: MappingNode,
	// 	  Style: "",
	// 	  Tag: (string) (len=5) "!!map",
	// 	  Value: (string) "",
	// 	  Content: ([]*yaml.Node) (len=2 cap=2) {
	// 	   (*yaml.Node)(0xc0000b8c80)({
	// 		Kind: ScalarNode,
	// 		Style: "",
	// 		Tag: (string) (len=5) "!!str",
	// 		Value: (string) (len=7) "creator",
	// 		Content: ([]*yaml.Node) <nil>,
	// 	   }),
	// 	   (*yaml.Node)(0xc0000b8d20)({
	// 		Kind: MappingNode,
	// 		Style: "",
	// 		Tag: (string) (len=5) "!!map",
	// 		Value: (string) "",
	// 		Content: ([]*yaml.Node) (len=4 cap=4) {
	// 		 (*yaml.Node)(0xc0000b8dc0)({
	// 		  Kind: ScalarNode,
	// 		  Style: "",
	// 		  Tag: (string) (len=5) "!!str",
	// 		  Value: (string) (len=4) "name",
	// 		  Content: ([]*yaml.Node) <nil>,
	// 		 }),
	// 		 (*yaml.Node)(0xc0000b8e60)({
	// 		  Kind: ScalarNode,
	// 		  Style: "",
	// 		  Tag: (string) (len=5) "!!str",
	// 		  Value: (string) (len=6) "Julien",
	// 		  Content: ([]*yaml.Node) <nil>,
	// 		 }),
	// 		 (*yaml.Node)(0xc0000b8f00)({
	// 		  Kind: ScalarNode,
	// 		  Style: "",
	// 		  Tag: (string) (len=5) "!!str",
	// 		  Value: (string) (len=7) "hobbies",
	// 		  Content: ([]*yaml.Node) <nil>,
	// 		 }),
	// 		 (*yaml.Node)(0xc0000b8fa0)({
	// 		  Kind: SequenceNode,
	// 		  Style: "",
	// 		  Tag: (string) (len=5) "!!seq",
	// 		  Value: (string) "",
	// 		  Content: ([]*yaml.Node) (len=1 cap=1) {
	// 		   (*yaml.Node)(0xc0000b9040)({
	// 			Kind: MappingNode,
	// 			Style: "",
	// 			Tag: (string) (len=5) "!!map",
	// 			Value: (string) "",
	// 			Content: ([]*yaml.Node) (len=4 cap=4) {
	// 			 (*yaml.Node)(0xc0000b90e0)({
	// 			  Kind: ScalarNode,
	// 			  Style: "",
	// 			  Tag: (string) (len=5) "!!str",
	// 			  Value: (string) (len=4) "name",
	// 			  Content: ([]*yaml.Node) <nil>,
	// 			 }),
	// 			 (*yaml.Node)(0xc0000b9180)({
	// 			  Kind: ScalarNode,
	// 			  Style: "",
	// 			  Tag: (string) (len=5) "!!str",
	// 			  Value: (string) (len=7) "Running",
	// 			  Content: ([]*yaml.Node) <nil>,
	// 			 }),
	// 			 (*yaml.Node)(0xc0000b9220)({
	// 			  Kind: ScalarNode,
	// 			  Style: "",
	// 			  Tag: (string) (len=5) "!!str",
	// 			  Value: (string) (len=9) "frequency",
	// 			  Content: ([]*yaml.Node) <nil>,
	// 			 }),
	// 			 (*yaml.Node)(0xc0000b92c0)({
	// 			  Kind: ScalarNode,
	// 			  Style: "",
	// 			  Tag: (string) (len=5) "!!str",
	// 			  Value: (string) (len=7) "3/weeks",
	// 			  Content: ([]*yaml.Node) <nil>,
	// 			 })
	// 			},
	// 		   })
	// 		  },
	// 		 })
	// 		},
	// 	   })
	// 	  },
	// 	 })
	// 	},
	// }
}
