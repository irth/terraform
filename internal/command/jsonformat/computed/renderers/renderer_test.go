package renderers

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/google/go-cmp/cmp"
	"github.com/mitchellh/colorstring"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/plans"
)

func TestRenderers_Human(t *testing.T) {
	colorize := colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: true,
	}

	tcs := map[string]struct {
		diff     computed.Diff
		expected string
		opts     computed.RenderHumanOpts
	}{
		"primitive_create": {
			diff: computed.Diff{
				Renderer: Primitive(nil, 1.0, cty.Number),
				Action:   plans.Create,
			},
			expected: "1",
		},
		"primitive_delete": {
			diff: computed.Diff{
				Renderer: Primitive(1.0, nil, cty.Number),
				Action:   plans.Delete,
			},
			expected: "1 -> null",
		},
		"primitive_delete_override": {
			diff: computed.Diff{
				Renderer: Primitive(1.0, nil, cty.Number),
				Action:   plans.Delete,
			},
			opts:     computed.RenderHumanOpts{OverrideNullSuffix: true},
			expected: "1",
		},
		"primitive_update_to_null": {
			diff: computed.Diff{
				Renderer: Primitive(1.0, nil, cty.Number),
				Action:   plans.Update,
			},
			expected: "1 -> null",
		},
		"primitive_update_from_null": {
			diff: computed.Diff{
				Renderer: Primitive(nil, 1.0, cty.Number),
				Action:   plans.Update,
			},
			expected: "null -> 1",
		},
		"primitive_update": {
			diff: computed.Diff{
				Renderer: Primitive(0.0, 1.0, cty.Number),
				Action:   plans.Update,
			},
			expected: "0 -> 1",
		},
		"primitive_update_replace": {
			diff: computed.Diff{
				Renderer: Primitive(0.0, 1.0, cty.Number),
				Action:   plans.Update,
				Replace:  true,
			},
			expected: "0 -> 1 # forces replacement",
		},
		"sensitive_update": {
			diff: computed.Diff{
				Renderer: Sensitive(computed.Diff{
					Renderer: Primitive(0.0, 1.0, cty.Number),
					Action:   plans.Update,
				}, true, true),
				Action: plans.Update,
			},
			expected: "(sensitive)",
		},
		"sensitive_update_replace": {
			diff: computed.Diff{
				Renderer: Sensitive(computed.Diff{
					Renderer: Primitive(0.0, 1.0, cty.Number),
					Action:   plans.Update,
					Replace:  true,
				}, true, true),
				Action:  plans.Update,
				Replace: true,
			},
			expected: "(sensitive) # forces replacement",
		},
		"computed_create": {
			diff: computed.Diff{
				Renderer: Unknown(computed.Diff{}),
				Action:   plans.Create,
			},
			expected: "(known after apply)",
		},
		"computed_update": {
			diff: computed.Diff{
				Renderer: Unknown(computed.Diff{
					Renderer: Primitive(0.0, nil, cty.Number),
					Action:   plans.Delete,
				}),
				Action: plans.Update,
			},
			expected: "0 -> (known after apply)",
		},
		"object_created": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{}),
				Action:   plans.Create,
			},
			expected: "{}",
		},
		"object_created_with_attributes": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(nil, 0.0, cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Create,
			},
			expected: `
{
      + attribute_one = 0
    }
`,
		},
		"object_deleted": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{}),
				Action:   plans.Delete,
			},
			expected: "{} -> null",
		},
		"object_deleted_with_attributes": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(0.0, nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Delete,
			},
			expected: `
{
      - attribute_one = 0
    } -> null
`,
		},
		"nested_object_deleted": {
			diff: computed.Diff{
				Renderer: NestedObject(map[string]computed.Diff{}),
				Action:   plans.Delete,
			},
			expected: "{} -> null",
		},
		"nested_object_deleted_with_attributes": {
			diff: computed.Diff{
				Renderer: NestedObject(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(0.0, nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Delete,
			},
			expected: `
{
      - attribute_one = 0 -> null
    } -> null
`,
		},
		"object_create_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(nil, 0.0, cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + attribute_one = 0
    }
`,
		},
		"object_update_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(0.0, 1.0, cty.Number),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ attribute_one = 0 -> 1
    }
`,
		},
		"object_update_attribute_forces_replacement": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(0.0, 1.0, cty.Number),
						Action:   plans.Update,
					},
				}),
				Action:  plans.Update,
				Replace: true,
			},
			expected: `
{ # forces replacement
      ~ attribute_one = 0 -> 1
    }
`,
		},
		"object_delete_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(0.0, nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      - attribute_one = 0
    }
`,
		},
		"object_ignore_unchanged_attributes": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(0.0, 1.0, cty.Number),
						Action:   plans.Update,
					},
					"attribute_two": {
						Renderer: Primitive(0.0, 0.0, cty.Number),
						Action:   plans.NoOp,
					},
					"attribute_three": {
						Renderer: Primitive(nil, 1.0, cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ attribute_one   = 0 -> 1
      + attribute_three = 1
        # (1 unchanged attribute hidden)
    }
`,
		},
		"object_create_sensitive_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(nil, 1.0, cty.Number),
							Action:   plans.Create,
						}, false, true),
						Action: plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + attribute_one = (sensitive)
    }
`,
		},
		"object_update_sensitive_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(0.0, 1.0, cty.Number),
							Action:   plans.Update,
						}, true, true),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ attribute_one = (sensitive)
    }
`,
		},
		"object_delete_sensitive_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(0.0, nil, cty.Number),
							Action:   plans.Delete,
						}, true, false),
						Action: plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      - attribute_one = (sensitive)
    }
`,
		},
		"object_create_computed_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Unknown(computed.Diff{Renderer: nil}),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + attribute_one = (known after apply)
    }
`,
		},
		"object_update_computed_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Unknown(computed.Diff{
							Renderer: Primitive(1.0, nil, cty.Number),
							Action:   plans.Delete,
						}),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ attribute_one = 1 -> (known after apply)
    }
`,
		},
		"object_escapes_attribute_keys": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(1.0, 2.0, cty.Number),
						Action:   plans.Update,
					},
					"attribute:two": {
						Renderer: Primitive(2.0, 3.0, cty.Number),
						Action:   plans.Update,
					},
					"attribute_six": {
						Renderer: Primitive(3.0, 4.0, cty.Number),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ "attribute:two" = 2 -> 3
      ~ attribute_one   = 1 -> 2
      ~ attribute_six   = 3 -> 4
    }
`,
		},
		"map_create_empty": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{}),
				Action:   plans.Create,
			},
			expected: "{}",
		},
		"map_create": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive(nil, "new", cty.String),
						Action:   plans.Create,
					},
				}),
				Action: plans.Create,
			},
			expected: `
{
      + "element_one" = "new"
    }
`,
		},
		"map_delete_empty": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{}),
				Action:   plans.Delete,
			},
			expected: "{} -> null",
		},
		"map_delete": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive("old", nil, cty.String),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Delete,
			},
			expected: `
{
      - "element_one" = "old"
    } -> null
`,
		},
		"map_create_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive(nil, "new", cty.String),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + "element_one" = "new"
    }
`,
		},
		"map_update_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive("old", "new", cty.String),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ "element_one" = "old" -> "new"
    }
`,
		},
		"map_delete_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive("old", nil, cty.String),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      - "element_one" = "old" -> null
    }
`,
		},
		"map_update_forces_replacement": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive("old", "new", cty.String),
						Action:   plans.Update,
					},
				}),
				Action:  plans.Update,
				Replace: true,
			},
			expected: `
{ # forces replacement
      ~ "element_one" = "old" -> "new"
    }
`,
		},
		"map_ignore_unchanged_elements": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive(nil, "new", cty.String),
						Action:   plans.Create,
					},
					"element_two": {
						Renderer: Primitive("old", "old", cty.String),
						Action:   plans.NoOp,
					},
					"element_three": {
						Renderer: Primitive("old", "new", cty.String),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + "element_one"   = "new"
      ~ "element_three" = "old" -> "new"
        # (1 unchanged element hidden)
    }
`,
		},
		"map_create_sensitive_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(nil, 1.0, cty.Number),
							Action:   plans.Create,
						}, false, true),
						Action: plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + "element_one" = (sensitive)
    }
`,
		},
		"map_update_sensitive_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(0.0, 1.0, cty.Number),
							Action:   plans.Update,
						}, true, true),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ "element_one" = (sensitive)
    }
`,
		},
		"map_update_sensitive_element_status": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(0.0, 0.0, cty.Number),
							Action:   plans.NoOp,
						}, true, false),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change. The value is unchanged.
      ~ "element_one" = (sensitive)
    }
`,
		},
		"map_delete_sensitive_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(0.0, nil, cty.Number),
							Action:   plans.Delete,
						}, true, false),
						Action: plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      - "element_one" = (sensitive) -> null
    }
`,
		},
		"map_create_computed_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Unknown(computed.Diff{}),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + "element_one" = (known after apply)
    }
`,
		},
		"map_update_computed_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Unknown(computed.Diff{
							Renderer: Primitive(1.0, nil, cty.Number),
							Action:   plans.Delete,
						}),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ "element_one" = 1 -> (known after apply)
    }
`,
		},
		"list_create_empty": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{}),
				Action:   plans.Create,
			},
			expected: "[]",
		},
		"list_create": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(nil, 1.0, cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Create,
			},
			expected: `
[
      + 1,
    ]
`,
		},
		"list_delete_empty": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{}),
				Action:   plans.Delete,
			},
			expected: "[] -> null",
		},
		"list_delete": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(1.0, nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Delete,
			},
			expected: `
[
      - 1,
    ] -> null
`,
		},
		"list_create_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(nil, 1.0, cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      + 1,
    ]
`,
		},
		"list_update_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(0.0, 1.0, cty.Number),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ 0 -> 1,
    ]
`,
		},
		"list_replace_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(0.0, nil, cty.Number),
						Action:   plans.Delete,
					},
					{
						Renderer: Primitive(nil, 1.0, cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      - 0,
      + 1,
    ]
`,
		},
		"list_delete_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(0.0, nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      - 0,
    ]
`,
		},
		"list_update_forces_replacement": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(0.0, 1.0, cty.Number),
						Action:   plans.Update,
					},
				}),
				Action:  plans.Update,
				Replace: true,
			},
			expected: `
[ # forces replacement
      ~ 0 -> 1,
    ]
`,
		},
		"list_update_ignores_unchanged": {
			diff: computed.Diff{
				Renderer: NestedList([]computed.Diff{
					{
						Renderer: Primitive(0.0, 0.0, cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(1.0, 1.0, cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(2.0, 5.0, cty.Number),
						Action:   plans.Update,
					},
					{
						Renderer: Primitive(3.0, 3.0, cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(4.0, 4.0, cty.Number),
						Action:   plans.NoOp,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ 2 -> 5,
        # (4 unchanged elements hidden)
    ]
`,
		},
		"list_update_ignored_unchanged_with_context": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(0.0, 0.0, cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(1.0, 1.0, cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(2.0, 5.0, cty.Number),
						Action:   plans.Update,
					},
					{
						Renderer: Primitive(3.0, 3.0, cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(4.0, 4.0, cty.Number),
						Action:   plans.NoOp,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
        # (1 unchanged element hidden)
        1,
      ~ 2 -> 5,
        3,
        # (1 unchanged element hidden)
    ]
`,
		},
		"list_create_sensitive_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(nil, 1.0, cty.Number),
							Action:   plans.Create,
						}, false, true),
						Action: plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      + (sensitive),
    ]
`,
		},
		"list_delete_sensitive_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(1.0, nil, cty.Number),
							Action:   plans.Delete,
						}, true, false),
						Action: plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      - (sensitive),
    ]
`,
		},
		"list_update_sensitive_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(0.0, 1.0, cty.Number),
							Action:   plans.Update,
						}, true, true),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ (sensitive),
    ]
`,
		},
		"list_update_sensitive_element_status": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(1.0, 1.0, cty.Number),
							Action:   plans.NoOp,
						}, false, true),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      # Warning: this attribute value will be marked as sensitive and will not
      # display in UI output after applying this change. The value is unchanged.
      ~ (sensitive),
    ]
`,
		},
		"list_create_computed_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Unknown(computed.Diff{}),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      + (known after apply),
    ]
`,
		},
		"list_update_computed_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Unknown(computed.Diff{
							Renderer: Primitive(0.0, nil, cty.Number),
							Action:   plans.Delete,
						}),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ 0 -> (known after apply),
    ]
`,
		},
		"set_create_empty": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{}),
				Action:   plans.Create,
			},
			expected: "[]",
		},
		"set_create": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(nil, 1.0, cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Create,
			},
			expected: `
[
      + 1,
    ]
`,
		},
		"set_delete_empty": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{}),
				Action:   plans.Delete,
			},
			expected: "[] -> null",
		},
		"set_delete": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(1.0, nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Delete,
			},
			expected: `
[
      - 1,
    ] -> null
`,
		},
		"set_create_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(nil, 1.0, cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      + 1,
    ]
`,
		},
		"set_update_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(0.0, 1.0, cty.Number),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ 0 -> 1,
    ]
`,
		},
		"set_replace_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(0.0, nil, cty.Number),
						Action:   plans.Delete,
					},
					{
						Renderer: Primitive(nil, 1.0, cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      - 0,
      + 1,
    ]
`,
		},
		"set_delete_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(0.0, nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      - 0,
    ]
`,
		},
		"set_update_forces_replacement": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(0.0, 1.0, cty.Number),
						Action:   plans.Update,
					},
				}),
				Action:  plans.Update,
				Replace: true,
			},
			expected: `
[ # forces replacement
      ~ 0 -> 1,
    ]
`,
		},
		"set_update_ignores_unchanged": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(0.0, 0.0, cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(1.0, 1.0, cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(2.0, 5.0, cty.Number),
						Action:   plans.Update,
					},
					{
						Renderer: Primitive(3.0, 3.0, cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(4.0, 4.0, cty.Number),
						Action:   plans.NoOp,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ 2 -> 5,
        # (4 unchanged elements hidden)
    ]
`,
		},
		"set_create_sensitive_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(nil, 1.0, cty.Number),
							Action:   plans.Create,
						}, false, true),
						Action: plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      + (sensitive),
    ]
`,
		},
		"set_delete_sensitive_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(1.0, nil, cty.Number),
							Action:   plans.Delete,
						}, false, true),
						Action: plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      - (sensitive),
    ]
`,
		},
		"set_update_sensitive_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(0.0, 1.0, cty.Number),
							Action:   plans.Update,
						}, true, true),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ (sensitive),
    ]
`,
		},
		"set_update_sensitive_element_status": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(1.0, 2.0, cty.Number),
							Action:   plans.Update,
						}, false, true),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      # Warning: this attribute value will be marked as sensitive and will not
      # display in UI output after applying this change.
      ~ (sensitive),
    ]
`,
		},
		"set_create_computed_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Unknown(computed.Diff{}),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      + (known after apply),
    ]
`,
		},
		"set_update_computed_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Unknown(computed.Diff{
							Renderer: Primitive(0.0, nil, cty.Number),
							Action:   plans.Delete,
						}),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ 0 -> (known after apply),
    ]
`,
		},
		"create_empty_block": {
			diff: computed.Diff{
				Renderer: Block(nil, nil),
				Action:   plans.Create,
			},
			expected: `
{
    }`,
		},
		"create_populated_block": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"string": {
						Renderer: Primitive(nil, "root", cty.String),
						Action:   plans.Create,
					},
					"boolean": {
						Renderer: Primitive(nil, true, cty.Bool),
						Action:   plans.Create,
					},
				}, map[string][]computed.Diff{
					"nested_block": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive(nil, "one", cty.String),
									Action:   plans.Create,
								},
							}, nil),
							Action: plans.Create,
						},
					},
					"nested_block_two": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive(nil, "two", cty.String),
									Action:   plans.Create,
								},
							}, nil),
							Action: plans.Create,
						},
					},
				}),
				Action: plans.Create,
			},
			expected: `
{
      + boolean = true
      + string  = "root"

      + nested_block {
          + string = "one"
        }

      + nested_block_two {
          + string = "two"
        }
    }`,
		},
		"update_empty_block": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"string": {
						Renderer: Primitive(nil, "root", cty.String),
						Action:   plans.Create,
					},
					"boolean": {
						Renderer: Primitive(nil, true, cty.Bool),
						Action:   plans.Create,
					},
				}, map[string][]computed.Diff{
					"nested_block": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive(nil, "one", cty.String),
									Action:   plans.Create,
								},
							}, nil),
							Action: plans.Create,
						},
					},
					"nested_block_two": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive(nil, "two", cty.String),
									Action:   plans.Create,
								},
							}, nil),
							Action: plans.Create,
						},
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + boolean = true
      + string  = "root"

      + nested_block {
          + string = "one"
        }

      + nested_block_two {
          + string = "two"
        }
    }`,
		},
		"update_populated_block": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"string": {
						Renderer: Primitive(nil, "root", cty.String),
						Action:   plans.Create,
					},
					"boolean": {
						Renderer: Primitive(false, true, cty.Bool),
						Action:   plans.Update,
					},
				}, map[string][]computed.Diff{
					"nested_block": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive(nil, "one", cty.String),
									Action:   plans.NoOp,
								},
							}, nil),
							Action: plans.NoOp,
						},
					},
					"nested_block_two": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive(nil, "two", cty.String),
									Action:   plans.Create,
								},
							}, nil),
							Action: plans.Create,
						},
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ boolean = false -> true
      + string  = "root"

      + nested_block_two {
          + string = "two"
        }
        # (1 unchanged block hidden)
    }`,
		},
		"clear_populated_block": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"string": {
						Renderer: Primitive("root", nil, cty.String),
						Action:   plans.Delete,
					},
					"boolean": {
						Renderer: Primitive(true, nil, cty.Bool),
						Action:   plans.Delete,
					},
				}, map[string][]computed.Diff{
					"nested_block": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("one", nil, cty.String),
									Action:   plans.Delete,
								},
							}, nil),
							Action: plans.Delete,
						},
					},
					"nested_block_two": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("two", nil, cty.String),
									Action:   plans.Delete,
								},
							}, nil),
							Action: plans.Delete,
						},
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      - boolean = true -> null
      - string  = "root" -> null

      - nested_block {
          - string = "one" -> null
        }

      - nested_block_two {
          - string = "two" -> null
        }
    }`,
		},
		"delete_populated_block": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"string": {
						Renderer: Primitive("root", nil, cty.String),
						Action:   plans.Delete,
					},
					"boolean": {
						Renderer: Primitive(true, nil, cty.Bool),
						Action:   plans.Delete,
					},
				}, map[string][]computed.Diff{
					"nested_block": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("one", nil, cty.String),
									Action:   plans.Delete,
								},
							}, nil),
							Action: plans.Delete,
						},
					},
					"nested_block_two": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("two", nil, cty.String),
									Action:   plans.Delete,
								},
							}, nil),
							Action: plans.Delete,
						},
					},
				}),
				Action: plans.Delete,
			},
			expected: `
{
      - boolean = true -> null
      - string  = "root" -> null

      - nested_block {
          - string = "one" -> null
        }

      - nested_block_two {
          - string = "two" -> null
        }
    }`,
		},
		"delete_empty_block": {
			diff: computed.Diff{
				Renderer: Block(nil, nil),
				Action:   plans.Delete,
			},
			expected: `
{
    }`,
		},
		"block_escapes_keys": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(1.0, 2.0, cty.Number),
						Action:   plans.Update,
					},
					"attribute:two": {
						Renderer: Primitive(2.0, 3.0, cty.Number),
						Action:   plans.Update,
					},
					"attribute_six": {
						Renderer: Primitive(3.0, 4.0, cty.Number),
						Action:   plans.Update,
					},
				}, map[string][]computed.Diff{
					"nested_block:one": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("one", "four", cty.String),
									Action:   plans.Update,
								},
							}, nil),
							Action: plans.Update,
						},
					},
					"nested_block_two": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("two", "three", cty.String),
									Action:   plans.Update,
								},
							}, nil),
							Action: plans.Update,
						},
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ "attribute:two" = 2 -> 3
      ~ attribute_one   = 1 -> 2
      ~ attribute_six   = 3 -> 4

      ~ "nested_block:one" {
          ~ string = "one" -> "four"
        }

      ~ nested_block_two {
          ~ string = "two" -> "three"
        }
    }`,
		},
		"block_always_includes_important_attributes": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"id": {
						Renderer: Primitive("root", "root", cty.String),
						Action:   plans.NoOp,
					},
					"boolean": {
						Renderer: Primitive(false, false, cty.Bool),
						Action:   plans.NoOp,
					},
				}, map[string][]computed.Diff{
					"nested_block": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("one", "one", cty.String),
									Action:   plans.NoOp,
								},
							}, nil),
							Action: plans.NoOp,
						},
					},
					"nested_block_two": {
						{
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("two", "two", cty.String),
									Action:   plans.NoOp,
								},
							}, nil),
							Action: plans.NoOp,
						},
					},
				}),
				Action: plans.NoOp,
			},
			expected: `
{
        id      = "root"
        # (1 unchanged attribute hidden)
        # (2 unchanged blocks hidden)
    }`,
		},
		"output_map_to_list": {
			diff: computed.Diff{
				Renderer: TypeChange(computed.Diff{
					Renderer: Map(map[string]computed.Diff{
						"element_one": {
							Renderer: Primitive(0.0, nil, cty.Number),
							Action:   plans.Delete,
						},
						"element_two": {
							Renderer: Primitive(1.0, nil, cty.Number),
							Action:   plans.Delete,
						},
					}),
					Action: plans.Delete,
				}, computed.Diff{
					Renderer: List([]computed.Diff{
						{
							Renderer: Primitive(nil, 0.0, cty.Number),
							Action:   plans.Create,
						},
						{
							Renderer: Primitive(nil, 1.0, cty.Number),
							Action:   plans.Create,
						},
					}),
					Action: plans.Create,
				}),
			},
			expected: `
{
      - "element_one" = 0
      - "element_two" = 1
    } -> [
      + 0,
      + 1,
    ]
`,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			expected := strings.TrimSpace(tc.expected)
			actual := colorize.Color(tc.diff.RenderHuman(0, tc.opts))
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Fatalf("\nexpected:\n%s\nactual:\n%s\ndiff:\n%s\n", expected, actual, diff)
			}
		})
	}

}