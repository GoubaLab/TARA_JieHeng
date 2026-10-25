// Purely synthetic, pre-migration Python domain oracle; no imported library records.
package assurance

const domainGoldenFixture = `{
  "validations": [
    {
      "action": "define-item",
      "payload": {},
      "expected": {
        "purpose": "",
        "boundary": "",
        "environment": "",
        "assumptions": "",
        "owner": "",
        "lifecycle": "概念阶段"
      }
    },
    {
      "action": "define-item",
      "payload": {
        "purpose": " \u001c功能　 "
      },
      "expected": {
        "purpose": "\u001c功能",
        "boundary": "",
        "environment": "",
        "assumptions": "",
        "owner": "",
        "lifecycle": "概念阶段"
      }
    },
    {
      "action": "define-item",
      "payload": {
        "other": 1
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {},
      "expected": {
        "nodes": [],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          }
        ]
      },
      "expected": {
        "nodes": [
          {
            "id": "ecu",
            "name": "ECU",
            "kind": "entity",
            "entityType": "ecu",
            "children": [],
            "attributes": [],
            "x": 0.0,
            "y": 1.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          }
        ],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "record-asset",
      "payload": {
        "name": "消息",
        "properties": [
          "integrity"
        ]
      },
      "expected": {
        "id": null,
        "name": "消息",
        "description": "",
        "node_ids": [],
        "properties": [
          "integrity"
        ],
        "reference": null
      }
    },
    {
      "action": "record-asset",
      "payload": {
        "name": 123,
        "properties": [
          "integrity"
        ]
      },
      "invalid": true
    },
    {
      "action": "record-scenario",
      "payload": {
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t"
      },
      "expected": {
        "id": null,
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t",
        "impact": {
          "safety": null,
          "financial": null,
          "operational": null,
          "privacy": null,
          "rationale": ""
        },
        "paths": [],
        "references": []
      }
    },
    {
      "action": "record-scenario",
      "payload": {
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t",
        "paths": [
          {
            "name": "路径"
          }
        ]
      },
      "expected": {
        "id": null,
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t",
        "impact": {
          "safety": null,
          "financial": null,
          "operational": null,
          "privacy": null,
          "rationale": ""
        },
        "paths": [
          {
            "id": "path_0000000000000001",
            "name": "路径",
            "steps": "",
            "feasibility": null,
            "rationale": "",
            "reference": null
          }
        ],
        "references": []
      }
    },
    {
      "action": "decide-treatment",
      "payload": {
        "scenario_id": "s",
        "treatment": "retain",
        "rationale": "r",
        "owner": "o"
      },
      "expected": {
        "scenario_id": "s",
        "treatment": "retain",
        "rationale": "r",
        "owner": "o",
        "goal": "",
        "claim": "",
        "residual_impact": null,
        "residual_feasibility": null,
        "residual_rationale": ""
      }
    },
    {
      "action": "specify-requirement",
      "payload": {
        "scenario_id": "s",
        "statement": "r",
        "allocation": "a",
        "verification": "v"
      },
      "expected": {
        "id": null,
        "scenario_id": "s",
        "statement": "r",
        "allocation": "a",
        "verification": "v",
        "standard_refs": [],
        "reference": null
      }
    },
    {
      "action": "select-regulations",
      "payload": {
        "references": [
          "reg:a:1",
          "reg:a:1",
          "reg:a:2"
        ]
      },
      "expected": {
        "references": [
          "reg:a:1",
          "reg:a:2"
        ]
      }
    },
    {
      "action": "record-evidence",
      "payload": {
        "requirement_id": "r",
        "result": "pass",
        "reference": "doc",
        "summary": "ok",
        "reviewer": "人"
      },
      "expected": {
        "id": null,
        "requirement_id": "r",
        "result": "pass",
        "reference": "doc",
        "summary": "ok",
        "reviewer": "人"
      }
    },
    {
      "action": "adopt-method",
      "payload": {
        "name": "组织示例方法 · 4 × 4",
        "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
        "matrix": [
          [
            1,
            1,
            1,
            1
          ],
          [
            1,
            2,
            2,
            3
          ],
          [
            2,
            3,
            3,
            4
          ],
          [
            3,
            4,
            5,
            5
          ]
        ]
      },
      "expected": {
        "name": "组织示例方法 · 4 × 4",
        "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
        "matrix": [
          [
            1,
            1,
            1,
            1
          ],
          [
            1,
            2,
            2,
            3
          ],
          [
            2,
            3,
            3,
            4
          ],
          [
            3,
            4,
            5,
            5
          ]
        ]
      }
    },
    {
      "action": "apply-reference-selection",
      "payload": {
        "asset_id": "a",
        "reference": "systems:123",
        "relationship_keys": [
          "relationships:1"
        ]
      },
      "expected": {
        "asset_id": "a",
        "reference": "systems:123",
        "relationship_keys": [
          "relationships:1"
        ],
        "include_controls": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "equal_spacing": true
      },
      "expected": {
        "nodes": [],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "equal_spacing": false
      },
      "expected": {
        "nodes": [],
        "flows": [],
        "equal_spacing": false
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "equal_spacing": 0
      },
      "expected": {
        "nodes": [],
        "flows": [],
        "equal_spacing": false
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "equal_spacing": 1
      },
      "expected": {
        "nodes": [],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "equal_spacing": 1.0
      },
      "expected": {
        "nodes": [],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "equal_spacing": "true"
      },
      "expected": {
        "nodes": [],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "equal_spacing": "TRUE"
      },
      "expected": {
        "nodes": [],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "equal_spacing": "1"
      },
      "expected": {
        "nodes": [],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "equal_spacing": "1.0"
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "equal_spacing": " true "
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "equal_spacing": null
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "equal_spacing": 2
      },
      "invalid": true
    },
    {
      "action": "record-scenario",
      "payload": {
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t",
        "impact": {
          "safety": 0
        }
      },
      "expected": {
        "id": null,
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t",
        "impact": {
          "safety": 0,
          "financial": null,
          "operational": null,
          "privacy": null,
          "rationale": ""
        },
        "paths": [],
        "references": []
      }
    },
    {
      "action": "record-scenario",
      "payload": {
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t",
        "impact": {
          "safety": 0.0
        }
      },
      "invalid": true
    },
    {
      "action": "record-scenario",
      "payload": {
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t",
        "impact": {
          "safety": true
        }
      },
      "invalid": true
    },
    {
      "action": "record-scenario",
      "payload": {
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t",
        "impact": {
          "safety": "2"
        }
      },
      "invalid": true
    },
    {
      "action": "record-scenario",
      "payload": {
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t",
        "impact": {
          "safety": null
        }
      },
      "expected": {
        "id": null,
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t",
        "impact": {
          "safety": null,
          "financial": null,
          "operational": null,
          "privacy": null,
          "rationale": ""
        },
        "paths": [],
        "references": []
      }
    },
    {
      "action": "record-scenario",
      "payload": {
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t",
        "impact": {
          "safety": 4
        }
      },
      "invalid": true
    },
    {
      "action": "record-scenario",
      "payload": {
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t",
        "impact": {
          "safety": -1
        }
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          }
        ]
      },
      "expected": {
        "nodes": [
          {
            "id": "ecu",
            "name": "ECU",
            "kind": "entity",
            "entityType": "ecu",
            "children": [],
            "attributes": [],
            "x": 0.0,
            "y": 1.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          }
        ],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 1.0,
            "y": 1.0
          }
        ]
      },
      "expected": {
        "nodes": [
          {
            "id": "ecu",
            "name": "ECU",
            "kind": "entity",
            "entityType": "ecu",
            "children": [],
            "attributes": [],
            "x": 1.0,
            "y": 1.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          }
        ],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": true,
            "y": 1.0
          }
        ]
      },
      "expected": {
        "nodes": [
          {
            "id": "ecu",
            "name": "ECU",
            "kind": "entity",
            "entityType": "ecu",
            "children": [],
            "attributes": [],
            "x": 1.0,
            "y": 1.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          }
        ],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": "1.2",
            "y": 1.0
          }
        ]
      },
      "expected": {
        "nodes": [
          {
            "id": "ecu",
            "name": "ECU",
            "kind": "entity",
            "entityType": "ecu",
            "children": [],
            "attributes": [],
            "x": 1.2,
            "y": 1.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          }
        ],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": "1_000",
            "y": 1.0
          }
        ]
      },
      "expected": {
        "nodes": [
          {
            "id": "ecu",
            "name": "ECU",
            "kind": "entity",
            "entityType": "ecu",
            "children": [],
            "attributes": [],
            "x": 1000.0,
            "y": 1.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          }
        ],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": "NaN",
            "y": 1.0
          }
        ]
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": null,
            "y": 1.0
          }
        ]
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 10001,
            "y": 1.0
          }
        ]
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          },
          {
            "id": "other",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          }
        ],
        "flows": [
          {
            "id": "f",
            "name": "f",
            "source": "ecu",
            "target": "other",
            "labelOffset": {
              "x": 0
            }
          }
        ]
      },
      "expected": {
        "nodes": [
          {
            "id": "ecu",
            "name": "ECU",
            "kind": "entity",
            "entityType": "ecu",
            "children": [],
            "attributes": [],
            "x": 0.0,
            "y": 1.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          },
          {
            "id": "other",
            "name": "ECU",
            "kind": "entity",
            "entityType": "ecu",
            "children": [],
            "attributes": [],
            "x": 0.0,
            "y": 1.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          }
        ],
        "flows": [
          {
            "id": "f",
            "name": "f",
            "source": "ecu",
            "target": "other",
            "sourceSide": null,
            "targetSide": null,
            "labelOffset": {
              "x": 0.0,
              "y": 0
            },
            "protocol": "",
            "description": ""
          }
        ],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          },
          {
            "id": "other",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          }
        ],
        "flows": [
          {
            "id": "f",
            "name": "f",
            "source": "ecu",
            "target": "other",
            "labelOffset": {
              "x": 1.0
            }
          }
        ]
      },
      "expected": {
        "nodes": [
          {
            "id": "ecu",
            "name": "ECU",
            "kind": "entity",
            "entityType": "ecu",
            "children": [],
            "attributes": [],
            "x": 0.0,
            "y": 1.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          },
          {
            "id": "other",
            "name": "ECU",
            "kind": "entity",
            "entityType": "ecu",
            "children": [],
            "attributes": [],
            "x": 0.0,
            "y": 1.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          }
        ],
        "flows": [
          {
            "id": "f",
            "name": "f",
            "source": "ecu",
            "target": "other",
            "sourceSide": null,
            "targetSide": null,
            "labelOffset": {
              "x": 1.0,
              "y": 0
            },
            "protocol": "",
            "description": ""
          }
        ],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          },
          {
            "id": "other",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          }
        ],
        "flows": [
          {
            "id": "f",
            "name": "f",
            "source": "ecu",
            "target": "other",
            "labelOffset": {
              "x": true
            }
          }
        ]
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          },
          {
            "id": "other",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          }
        ],
        "flows": [
          {
            "id": "f",
            "name": "f",
            "source": "ecu",
            "target": "other",
            "labelOffset": {
              "x": "1.2"
            }
          }
        ]
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          },
          {
            "id": "other",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          }
        ],
        "flows": [
          {
            "id": "f",
            "name": "f",
            "source": "ecu",
            "target": "other",
            "labelOffset": {
              "x": null
            }
          }
        ]
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          },
          {
            "id": "other",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          }
        ],
        "flows": [
          {
            "id": "f",
            "name": "f",
            "source": "ecu",
            "target": "other",
            "labelOffset": {
              "x": 10001
            }
          }
        ]
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0,
            "attributes": [
              {
                "name": "Straße"
              },
              {
                "name": "STRASSE"
              }
            ]
          }
        ]
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0,
            "attributes": [
              {
                "name": "Σ"
              },
              {
                "name": "ς"
              }
            ]
          }
        ]
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0,
            "attributes": [
              {
                "name": "电压",
                "value": "400V"
              }
            ]
          }
        ]
      },
      "expected": {
        "nodes": [
          {
            "id": "ecu",
            "name": "ECU",
            "kind": "entity",
            "entityType": "ecu",
            "children": [],
            "attributes": [
              {
                "name": "电压",
                "value": "400V"
              }
            ],
            "x": 0.0,
            "y": 1.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          }
        ],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "apply-reference-selection",
      "payload": {
        "asset_id": "a",
        "reference": "systems:123",
        "relationship_keys": [
          "relationships:1"
        ],
        "include_controls": true
      },
      "expected": {
        "asset_id": "a",
        "reference": "systems:123",
        "relationship_keys": [
          "relationships:1"
        ],
        "include_controls": true
      }
    },
    {
      "action": "apply-reference-selection",
      "payload": {
        "asset_id": "a",
        "reference": "systems:123",
        "relationship_keys": [
          "relationships:1"
        ],
        "include_controls": false
      },
      "expected": {
        "asset_id": "a",
        "reference": "systems:123",
        "relationship_keys": [
          "relationships:1"
        ],
        "include_controls": false
      }
    },
    {
      "action": "apply-reference-selection",
      "payload": {
        "asset_id": "a",
        "reference": "systems:123",
        "relationship_keys": [
          "relationships:1"
        ],
        "include_controls": "true"
      },
      "invalid": true
    },
    {
      "action": "apply-reference-selection",
      "payload": {
        "asset_id": "a",
        "reference": "systems:123",
        "relationship_keys": [
          "relationships:1"
        ],
        "include_controls": 1
      },
      "invalid": true
    },
    {
      "action": "apply-reference-selection",
      "payload": {
        "asset_id": "a",
        "reference": "systems:123",
        "relationship_keys": [
          "relationships:1"
        ],
        "include_controls": null
      },
      "invalid": true
    },
    {
      "action": "record-asset",
      "payload": {
        "name": "a",
        "properties": [
          "integrity"
        ]
      },
      "expected": {
        "id": null,
        "name": "a",
        "description": "",
        "node_ids": [],
        "properties": [
          "integrity"
        ],
        "reference": null
      }
    },
    {
      "action": "record-asset",
      "payload": {
        "name": "a",
        "properties": [
          " integrity "
        ]
      },
      "invalid": true
    },
    {
      "action": "record-asset",
      "payload": {
        "name": "a",
        "properties": [
          123
        ]
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0,
            "children": [
              {
                "id": "sw",
                "name": "软件",
                "kind": "software"
              }
            ]
          }
        ]
      },
      "expected": {
        "nodes": [
          {
            "id": "ecu",
            "name": "ECU",
            "kind": "entity",
            "entityType": "ecu",
            "children": [
              {
                "id": "sw",
                "name": "软件",
                "kind": "software",
                "description": "",
                "reference": null
              }
            ],
            "attributes": [],
            "x": 0.0,
            "y": 1.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          }
        ],
        "flows": [],
        "equal_spacing": true
      }
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "boundary",
            "x": 0,
            "y": 1.0,
            "children": [
              {
                "id": "sw",
                "name": "软件",
                "kind": "software"
              }
            ]
          }
        ]
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "store",
            "x": 0,
            "y": 1.0
          }
        ]
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0,
            "entityType": null
          }
        ]
      },
      "invalid": true
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0,
            "children": [
              {
                "id": "ecu",
                "name": "dup",
                "kind": "store"
              }
            ]
          }
        ]
      },
      "invalid": true
    },
    {
      "action": "adopt-method",
      "payload": {
        "name": "组织示例方法 · 4 × 4",
        "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
        "matrix": [
          [
            1,
            1,
            1,
            1
          ],
          [
            1,
            2,
            2,
            3
          ],
          [
            2,
            3,
            3,
            4
          ],
          [
            3,
            4,
            5,
            true
          ]
        ]
      },
      "invalid": true
    },
    {
      "action": "adopt-method",
      "payload": {
        "name": "组织示例方法 · 4 × 4",
        "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
        "matrix": [
          [
            1,
            1,
            1,
            1
          ],
          [
            1,
            2,
            2,
            3
          ],
          [
            2,
            3,
            3,
            4
          ],
          [
            3,
            4,
            5,
            4
          ]
        ]
      },
      "invalid": true
    },
    {
      "action": "record-scenario",
      "payload": {
        "title": "t",
        "asset_id": "a",
        "property": "integrity",
        "damage": "d",
        "threat": "t",
        "paths": [
          {
            "id": "p",
            "name": "p"
          },
          {
            "id": "p",
            "name": "p"
          }
        ]
      },
      "invalid": true
    }
  ],
  "digests": [
    {
      "value": null,
      "canonical": "null",
      "digest": "74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b"
    },
    {
      "value": true,
      "canonical": "true",
      "digest": "b5bea41b6c623f7c09f1bf24dcae58ebab3c0cdd90ad966bc43a45b44867e12b"
    },
    {
      "value": false,
      "canonical": "false",
      "digest": "fcbcf165908dd18a9e49f7ff27810176db8e9f63b4352213741664245224f8aa"
    },
    {
      "value": 0,
      "canonical": "0",
      "digest": "5feceb66ffc86f38d952786c6d696c79c2dbc239dd4e91b46729d73a27fb57e9"
    },
    {
      "value": 1,
      "canonical": "1",
      "digest": "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
    },
    {
      "value": -1,
      "canonical": "-1",
      "digest": "1bad6b8cf97131fceab8543e81f7757195fbb1d36b376ee994ad1cf17699c464"
    },
    {
      "value": 0.0,
      "canonical": "0.0",
      "digest": "8aed642bf5118b9d3c859bd4be35ecac75b6e873cce34e7b6f554b06f75550d7"
    },
    {
      "value": -0.0,
      "canonical": "-0.0",
      "digest": "c26617c7ccbcaa6631b45d851b8cf56e21d2ca624bdb1193afdbd4b560702cec"
    },
    {
      "value": 1.0,
      "canonical": "1.0",
      "digest": "d0ff5974b6aa52cf562bea5921840c032a860a91a3512f7fe8f768f6bbe005f6"
    },
    {
      "value": 0.0001,
      "canonical": "0.0001",
      "digest": "e539aebac7fd2ed8aba8854fedeaa65436962f8ee1c82341b394698f085b6496"
    },
    {
      "value": 1e-05,
      "canonical": "1e-05",
      "digest": "5a9530d3c6d5faad6ba8afe16fb8e438df23865128370b2ec14ef622bb5f2ec9"
    },
    {
      "value": 1000000.0,
      "canonical": "1000000.0",
      "digest": "73b65d062c013732e8221e2259178fb9da7c8e36faf15d7117f7e57619653f52"
    },
    {
      "value": 1000000000000000.0,
      "canonical": "1000000000000000.0",
      "digest": "c02731631f61648f06ba5501a9c68e2b0a9186a934ff45f40ab28eb6f3673021"
    },
    {
      "value": 1e+16,
      "canonical": "1e+16",
      "digest": "a144838520595009e7daf5aff8472573f9ce6a4bcd8c30673675618883464ab0"
    },
    {
      "value": 1e+20,
      "canonical": "1e+20",
      "digest": "7c18c9fbdcc8281573e9db9e04f04c3790b10696f3706f0f03fa87427d33e28b"
    },
    {
      "value": 1e+21,
      "canonical": "1e+21",
      "digest": "241c4643fa70b1dcde1205b71be4e3bebb17e9f880c8e1a33d0ead6c27271d3c"
    },
    {
      "value": 1e-308,
      "canonical": "1e-308",
      "digest": "b77bea380df569401be33b47bf3a5868d224bf2ab22e3cda531409a63edc9683"
    },
    {
      "value": 5e-324,
      "canonical": "5e-324",
      "digest": "c46e7ca1be4c8734f373a56530787288fa2058d73d07855e9247e949f811a42a"
    },
    {
      "value": 1.7976931348623157e+308,
      "canonical": "1.7976931348623157e+308",
      "digest": "c2784e1abd6317452708f3fbf9641c16b959561bc621a1d408c23a20aa2cb585"
    },
    {
      "value": 1267650600228229401496703205376,
      "canonical": "1267650600228229401496703205376",
      "digest": "5ee7924f903dfefa9253b5f63147f53433516da9daf01f6e729331dcf31b28c6"
    },
    {
      "value": {
        "z": "中文<&>  \n\b\f\r\t\u0000",
        "a": [
          "é",
          "𐀀",
          "￿"
        ]
      },
      "canonical": "{\"a\": [\"é\", \"𐀀\", \"￿\"], \"z\": \"中文<&>  \\n\\b\\f\\r\\t\\u0000\"}",
      "digest": "25686d440b0513fea818bc1cc0a0d3a5cd895ff5f99fa737200e44f776181dab"
    },
    {
      "value": [
        1.0000000000000001e+18,
        1.2345678901234567,
        1.0000000000000002
      ],
      "canonical": "[1.0000000000000001e+18, 1.2345678901234567, 1.0000000000000002]",
      "digest": "a2e500f78c93ac076ea91b765617dbdcff43e1a8a7ebea1f3b053c8b7cea6775"
    }
  ],
  "initial": {
    "id": "ws_0000000000000001",
    "title": "Golden 项目",
    "description": "测试历史兼容",
    "revision": 0,
    "created_at": "2026-10-25T12:34:56.123456+00:00",
    "updated_at": "2026-10-25T12:34:56.123456+00:00",
    "item": {
      "purpose": "",
      "boundary": "",
      "environment": "",
      "assumptions": "",
      "owner": "",
      "lifecycle": "概念阶段"
    },
    "architecture": {
      "nodes": [],
      "flows": [],
      "equal_spacing": true
    },
    "assets": [],
    "scenarios": [],
    "decisions": [],
    "requirements": [],
    "evidence": [],
    "method": {
      "name": "组织示例方法 · 4 × 4",
      "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
      "matrix": [
        [
          1,
          1,
          1,
          1
        ],
        [
          1,
          2,
          2,
          3
        ],
        [
          2,
          3,
          3,
          4
        ],
        [
          3,
          4,
          5,
          5
        ]
      ]
    },
    "method_confirmed": false,
    "baselines": [],
    "reference_imports": {},
    "regulation_refs": []
  },
  "initial_review": {
    "ready": false,
    "issues": [
      {
        "stage": "definition",
        "message": "补全功能、边界、运行环境、假设和责任人",
        "reference": null
      },
      {
        "stage": "assessment",
        "message": "确认组织采用的风险评估方法",
        "reference": null
      },
      {
        "stage": "architecture",
        "message": "建立系统架构并识别受保护资产",
        "reference": null
      },
      {
        "stage": "assessment",
        "message": "尚未建立损害场景、威胁及攻击路径",
        "reference": null
      }
    ],
    "evaluations": [],
    "requirements": [],
    "stages": [
      {
        "key": "definition",
        "complete": false
      },
      {
        "key": "architecture",
        "complete": false
      },
      {
        "key": "assessment",
        "complete": false
      },
      {
        "key": "treatment",
        "complete": false
      },
      {
        "key": "verification",
        "complete": false
      }
    ]
  },
  "steps": [
    {
      "action": "define-item",
      "payload": {
        "purpose": "功能",
        "boundary": "边界",
        "environment": "环境",
        "assumptions": "假设",
        "owner": "负责人"
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 1,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [],
          "flows": [],
          "equal_spacing": true
        },
        "assets": [],
        "scenarios": [],
        "decisions": [],
        "requirements": [],
        "evidence": [],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": false,
        "baselines": [],
        "reference_imports": {},
        "regulation_refs": []
      },
      "entity_id": "ws_0000000000000001",
      "review": {
        "ready": false,
        "issues": [
          {
            "stage": "assessment",
            "message": "确认组织采用的风险评估方法",
            "reference": null
          },
          {
            "stage": "architecture",
            "message": "建立系统架构并识别受保护资产",
            "reference": null
          },
          {
            "stage": "assessment",
            "message": "尚未建立损害场景、威胁及攻击路径",
            "reference": null
          }
        ],
        "evaluations": [],
        "requirements": [],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": false
          },
          {
            "key": "assessment",
            "complete": false
          },
          {
            "key": "treatment",
            "complete": false
          },
          {
            "key": "verification",
            "complete": false
          }
        ]
      },
      "digest": "84589cc4153bf427fac6c05d1f5ec06c5461c1d34f0a2a3a73e0e9e6343cec9c",
      "before_digest": "e4364193c2a2e51f94d5189f80d995088b574dc81bfdc9c1f5a8025d63e58c6d"
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": " ECU ",
            "kind": "process",
            "x": 0,
            "y": 1.0
          },
          {
            "id": "tester",
            "name": "终端",
            "kind": "external",
            "x": 400,
            "y": 0
          }
        ],
        "flows": [
          {
            "id": "can",
            "name": "消息",
            "source": "ecu",
            "target": "tester"
          }
        ]
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 2,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [
            {
              "id": "ecu",
              "name": "ECU",
              "kind": "entity",
              "entityType": "ecu",
              "children": [],
              "attributes": [],
              "x": 0.0,
              "y": 1.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            },
            {
              "id": "tester",
              "name": "终端",
              "kind": "entity",
              "entityType": "external",
              "children": [],
              "attributes": [],
              "x": 400.0,
              "y": 0.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            }
          ],
          "flows": [
            {
              "id": "can",
              "name": "消息",
              "source": "ecu",
              "target": "tester",
              "sourceSide": null,
              "targetSide": null,
              "labelOffset": {
                "x": 0,
                "y": 0
              },
              "protocol": "",
              "description": ""
            }
          ],
          "equal_spacing": true
        },
        "assets": [],
        "scenarios": [],
        "decisions": [],
        "requirements": [],
        "evidence": [],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": false,
        "baselines": [],
        "reference_imports": {},
        "regulation_refs": []
      },
      "entity_id": "ws_0000000000000001",
      "review": {
        "ready": false,
        "issues": [
          {
            "stage": "assessment",
            "message": "确认组织采用的风险评估方法",
            "reference": null
          },
          {
            "stage": "architecture",
            "message": "建立系统架构并识别受保护资产",
            "reference": null
          },
          {
            "stage": "assessment",
            "message": "尚未建立损害场景、威胁及攻击路径",
            "reference": null
          }
        ],
        "evaluations": [],
        "requirements": [],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": false
          },
          {
            "key": "assessment",
            "complete": false
          },
          {
            "key": "treatment",
            "complete": false
          },
          {
            "key": "verification",
            "complete": false
          }
        ]
      },
      "digest": "0e1cdfa57ee9a5421883a4e57264bf3858dc2db0a0372c8bfd58217b7e85cd10",
      "before_digest": "84589cc4153bf427fac6c05d1f5ec06c5461c1d34f0a2a3a73e0e9e6343cec9c"
    },
    {
      "action": "adopt-method",
      "payload": {
        "name": "组织示例方法 · 4 × 4",
        "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
        "matrix": [
          [
            1,
            1,
            1,
            1
          ],
          [
            1,
            2,
            2,
            3
          ],
          [
            2,
            3,
            3,
            4
          ],
          [
            3,
            4,
            5,
            5
          ]
        ]
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 3,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [
            {
              "id": "ecu",
              "name": "ECU",
              "kind": "entity",
              "entityType": "ecu",
              "children": [],
              "attributes": [],
              "x": 0.0,
              "y": 1.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            },
            {
              "id": "tester",
              "name": "终端",
              "kind": "entity",
              "entityType": "external",
              "children": [],
              "attributes": [],
              "x": 400.0,
              "y": 0.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            }
          ],
          "flows": [
            {
              "id": "can",
              "name": "消息",
              "source": "ecu",
              "target": "tester",
              "sourceSide": null,
              "targetSide": null,
              "labelOffset": {
                "x": 0,
                "y": 0
              },
              "protocol": "",
              "description": ""
            }
          ],
          "equal_spacing": true
        },
        "assets": [],
        "scenarios": [],
        "decisions": [],
        "requirements": [],
        "evidence": [],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": true,
        "baselines": [],
        "reference_imports": {},
        "regulation_refs": []
      },
      "entity_id": "ws_0000000000000001",
      "review": {
        "ready": false,
        "issues": [
          {
            "stage": "architecture",
            "message": "建立系统架构并识别受保护资产",
            "reference": null
          },
          {
            "stage": "assessment",
            "message": "尚未建立损害场景、威胁及攻击路径",
            "reference": null
          }
        ],
        "evaluations": [],
        "requirements": [],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": false
          },
          {
            "key": "assessment",
            "complete": false
          },
          {
            "key": "treatment",
            "complete": false
          },
          {
            "key": "verification",
            "complete": false
          }
        ]
      },
      "digest": "597b7c68b992cafb7ebd71139f61ffc56a1305fac5e7749185d41c440bf84c7d",
      "before_digest": "0e1cdfa57ee9a5421883a4e57264bf3858dc2db0a0372c8bfd58217b7e85cd10"
    },
    {
      "action": "record-asset",
      "payload": {
        "name": "控制指令",
        "node_ids": [
          "ecu"
        ],
        "properties": [
          "integrity"
        ]
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 4,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [
            {
              "id": "ecu",
              "name": "ECU",
              "kind": "entity",
              "entityType": "ecu",
              "children": [],
              "attributes": [],
              "x": 0.0,
              "y": 1.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            },
            {
              "id": "tester",
              "name": "终端",
              "kind": "entity",
              "entityType": "external",
              "children": [],
              "attributes": [],
              "x": 400.0,
              "y": 0.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            }
          ],
          "flows": [
            {
              "id": "can",
              "name": "消息",
              "source": "ecu",
              "target": "tester",
              "sourceSide": null,
              "targetSide": null,
              "labelOffset": {
                "x": 0,
                "y": 0
              },
              "protocol": "",
              "description": ""
            }
          ],
          "equal_spacing": true
        },
        "assets": [
          {
            "id": "asset_0000000000000002",
            "name": "控制指令",
            "description": "",
            "node_ids": [
              "ecu"
            ],
            "properties": [
              "integrity"
            ],
            "reference": null
          }
        ],
        "scenarios": [],
        "decisions": [],
        "requirements": [],
        "evidence": [],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": true,
        "baselines": [],
        "reference_imports": {},
        "regulation_refs": []
      },
      "entity_id": "asset_0000000000000002",
      "review": {
        "ready": false,
        "issues": [
          {
            "stage": "assessment",
            "message": "资产“控制指令”的 integrity 尚无风险场景",
            "reference": "asset_0000000000000002"
          },
          {
            "stage": "assessment",
            "message": "尚未建立损害场景、威胁及攻击路径",
            "reference": null
          }
        ],
        "evaluations": [],
        "requirements": [],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": true
          },
          {
            "key": "assessment",
            "complete": false
          },
          {
            "key": "treatment",
            "complete": false
          },
          {
            "key": "verification",
            "complete": false
          }
        ]
      },
      "digest": "e5e906a4b9c57602d82806df1ca2f574e0593ddf5eb8c01bd574c4877e9eefac",
      "before_digest": "597b7c68b992cafb7ebd71139f61ffc56a1305fac5e7749185d41c440bf84c7d"
    },
    {
      "action": "record-scenario",
      "payload": {
        "title": "篡改",
        "asset_id": "asset_0000000000000002",
        "property": "integrity",
        "damage": "异常",
        "threat": "注入",
        "impact": {
          "safety": 3,
          "financial": 1,
          "operational": 2,
          "privacy": 0,
          "rationale": "影响依据"
        },
        "paths": [
          {
            "name": "接入",
            "steps": "接入后发送",
            "feasibility": 3,
            "rationale": "测试"
          }
        ]
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 5,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [
            {
              "id": "ecu",
              "name": "ECU",
              "kind": "entity",
              "entityType": "ecu",
              "children": [],
              "attributes": [],
              "x": 0.0,
              "y": 1.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            },
            {
              "id": "tester",
              "name": "终端",
              "kind": "entity",
              "entityType": "external",
              "children": [],
              "attributes": [],
              "x": 400.0,
              "y": 0.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            }
          ],
          "flows": [
            {
              "id": "can",
              "name": "消息",
              "source": "ecu",
              "target": "tester",
              "sourceSide": null,
              "targetSide": null,
              "labelOffset": {
                "x": 0,
                "y": 0
              },
              "protocol": "",
              "description": ""
            }
          ],
          "equal_spacing": true
        },
        "assets": [
          {
            "id": "asset_0000000000000002",
            "name": "控制指令",
            "description": "",
            "node_ids": [
              "ecu"
            ],
            "properties": [
              "integrity"
            ],
            "reference": null
          }
        ],
        "scenarios": [
          {
            "id": "scenario_0000000000000004",
            "title": "篡改",
            "asset_id": "asset_0000000000000002",
            "property": "integrity",
            "damage": "异常",
            "threat": "注入",
            "impact": {
              "safety": 3,
              "financial": 1,
              "operational": 2,
              "privacy": 0,
              "rationale": "影响依据"
            },
            "paths": [
              {
                "id": "path_0000000000000003",
                "name": "接入",
                "steps": "接入后发送",
                "feasibility": 3,
                "rationale": "测试",
                "reference": null
              }
            ],
            "references": []
          }
        ],
        "decisions": [],
        "requirements": [],
        "evidence": [],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": true,
        "baselines": [],
        "reference_imports": {},
        "regulation_refs": []
      },
      "entity_id": "scenario_0000000000000004",
      "review": {
        "ready": false,
        "issues": [
          {
            "stage": "treatment",
            "message": "尚未记录风险处置决策",
            "reference": "scenario_0000000000000004"
          }
        ],
        "evaluations": [
          {
            "scenario_id": "scenario_0000000000000004",
            "impact": 3,
            "feasibility": 3,
            "level": 5,
            "paths": [
              {
                "id": "path_0000000000000003",
                "level": 5
              }
            ],
            "decision_stale": false,
            "residual": null
          }
        ],
        "requirements": [],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": true
          },
          {
            "key": "assessment",
            "complete": true
          },
          {
            "key": "treatment",
            "complete": false
          },
          {
            "key": "verification",
            "complete": false
          }
        ]
      },
      "digest": "7d90559d7dcc601913f396896eadfc04dfb4cc2c204ea24bfea9a24c0d51a718",
      "before_digest": "e5e906a4b9c57602d82806df1ca2f574e0593ddf5eb8c01bd574c4877e9eefac"
    },
    {
      "action": "decide-treatment",
      "payload": {
        "scenario_id": "scenario_0000000000000004",
        "treatment": "reduce",
        "owner": "负责人",
        "rationale": "分析",
        "goal": "验证消息",
        "residual_impact": 2,
        "residual_feasibility": 1,
        "residual_rationale": "独立评估"
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 6,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [
            {
              "id": "ecu",
              "name": "ECU",
              "kind": "entity",
              "entityType": "ecu",
              "children": [],
              "attributes": [],
              "x": 0.0,
              "y": 1.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            },
            {
              "id": "tester",
              "name": "终端",
              "kind": "entity",
              "entityType": "external",
              "children": [],
              "attributes": [],
              "x": 400.0,
              "y": 0.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            }
          ],
          "flows": [
            {
              "id": "can",
              "name": "消息",
              "source": "ecu",
              "target": "tester",
              "sourceSide": null,
              "targetSide": null,
              "labelOffset": {
                "x": 0,
                "y": 0
              },
              "protocol": "",
              "description": ""
            }
          ],
          "equal_spacing": true
        },
        "assets": [
          {
            "id": "asset_0000000000000002",
            "name": "控制指令",
            "description": "",
            "node_ids": [
              "ecu"
            ],
            "properties": [
              "integrity"
            ],
            "reference": null
          }
        ],
        "scenarios": [
          {
            "id": "scenario_0000000000000004",
            "title": "篡改",
            "asset_id": "asset_0000000000000002",
            "property": "integrity",
            "damage": "异常",
            "threat": "注入",
            "impact": {
              "safety": 3,
              "financial": 1,
              "operational": 2,
              "privacy": 0,
              "rationale": "影响依据"
            },
            "paths": [
              {
                "id": "path_0000000000000003",
                "name": "接入",
                "steps": "接入后发送",
                "feasibility": 3,
                "rationale": "测试",
                "reference": null
              }
            ],
            "references": []
          }
        ],
        "decisions": [
          {
            "scenario_id": "scenario_0000000000000004",
            "treatment": "reduce",
            "rationale": "分析",
            "owner": "负责人",
            "goal": "验证消息",
            "claim": "",
            "residual_impact": 2,
            "residual_feasibility": 1,
            "residual_rationale": "独立评估",
            "assessment_hash": "37ad7a31386fbbf3902d94cb4ae48f9cdb256bd732d5eea601094edcaa6ccb00",
            "decided_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "requirements": [],
        "evidence": [],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": true,
        "baselines": [],
        "reference_imports": {},
        "regulation_refs": []
      },
      "entity_id": "scenario_0000000000000004",
      "review": {
        "ready": false,
        "issues": [
          {
            "stage": "treatment",
            "message": "降低风险需要安全目标和分配到组件的安全需求",
            "reference": "scenario_0000000000000004"
          }
        ],
        "evaluations": [
          {
            "scenario_id": "scenario_0000000000000004",
            "impact": 3,
            "feasibility": 3,
            "level": 5,
            "paths": [
              {
                "id": "path_0000000000000003",
                "level": 5
              }
            ],
            "decision_stale": false,
            "residual": 2
          }
        ],
        "requirements": [],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": true
          },
          {
            "key": "assessment",
            "complete": true
          },
          {
            "key": "treatment",
            "complete": false
          },
          {
            "key": "verification",
            "complete": false
          }
        ]
      },
      "digest": "7ca25e67cd6aa14a8ca242a31d7c0c9c2a5a25097764403aa3ce677f3d21f003",
      "before_digest": "7d90559d7dcc601913f396896eadfc04dfb4cc2c204ea24bfea9a24c0d51a718"
    },
    {
      "action": "specify-requirement",
      "payload": {
        "scenario_id": "scenario_0000000000000004",
        "statement": "验证来源",
        "allocation": "ECU",
        "verification": "篡改拒绝"
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 7,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [
            {
              "id": "ecu",
              "name": "ECU",
              "kind": "entity",
              "entityType": "ecu",
              "children": [],
              "attributes": [],
              "x": 0.0,
              "y": 1.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            },
            {
              "id": "tester",
              "name": "终端",
              "kind": "entity",
              "entityType": "external",
              "children": [],
              "attributes": [],
              "x": 400.0,
              "y": 0.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            }
          ],
          "flows": [
            {
              "id": "can",
              "name": "消息",
              "source": "ecu",
              "target": "tester",
              "sourceSide": null,
              "targetSide": null,
              "labelOffset": {
                "x": 0,
                "y": 0
              },
              "protocol": "",
              "description": ""
            }
          ],
          "equal_spacing": true
        },
        "assets": [
          {
            "id": "asset_0000000000000002",
            "name": "控制指令",
            "description": "",
            "node_ids": [
              "ecu"
            ],
            "properties": [
              "integrity"
            ],
            "reference": null
          }
        ],
        "scenarios": [
          {
            "id": "scenario_0000000000000004",
            "title": "篡改",
            "asset_id": "asset_0000000000000002",
            "property": "integrity",
            "damage": "异常",
            "threat": "注入",
            "impact": {
              "safety": 3,
              "financial": 1,
              "operational": 2,
              "privacy": 0,
              "rationale": "影响依据"
            },
            "paths": [
              {
                "id": "path_0000000000000003",
                "name": "接入",
                "steps": "接入后发送",
                "feasibility": 3,
                "rationale": "测试",
                "reference": null
              }
            ],
            "references": []
          }
        ],
        "decisions": [
          {
            "scenario_id": "scenario_0000000000000004",
            "treatment": "reduce",
            "rationale": "分析",
            "owner": "负责人",
            "goal": "验证消息",
            "claim": "",
            "residual_impact": 2,
            "residual_feasibility": 1,
            "residual_rationale": "独立评估",
            "assessment_hash": "37ad7a31386fbbf3902d94cb4ae48f9cdb256bd732d5eea601094edcaa6ccb00",
            "decided_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "requirements": [
          {
            "id": "req_0000000000000005",
            "scenario_id": "scenario_0000000000000004",
            "statement": "验证来源",
            "allocation": "ECU",
            "verification": "篡改拒绝",
            "standard_refs": [],
            "reference": null
          }
        ],
        "evidence": [],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": true,
        "baselines": [],
        "reference_imports": {},
        "regulation_refs": []
      },
      "entity_id": "req_0000000000000005",
      "review": {
        "ready": false,
        "issues": [
          {
            "stage": "verification",
            "message": "安全需求缺少有效的通过验证记录",
            "reference": "req_0000000000000005"
          }
        ],
        "evaluations": [
          {
            "scenario_id": "scenario_0000000000000004",
            "impact": 3,
            "feasibility": 3,
            "level": 5,
            "paths": [
              {
                "id": "path_0000000000000003",
                "level": 5
              }
            ],
            "decision_stale": false,
            "residual": 2
          }
        ],
        "requirements": [
          {
            "requirement_id": "req_0000000000000005",
            "status": "missing",
            "draft": false,
            "latest_evidence_id": null
          }
        ],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": true
          },
          {
            "key": "assessment",
            "complete": true
          },
          {
            "key": "treatment",
            "complete": true
          },
          {
            "key": "verification",
            "complete": false
          }
        ]
      },
      "digest": "cf39899f8491785ee203c7f74eac912f85854b188016bd06b05951a4ca65d136",
      "before_digest": "7ca25e67cd6aa14a8ca242a31d7c0c9c2a5a25097764403aa3ce677f3d21f003"
    },
    {
      "action": "record-evidence",
      "payload": {
        "requirement_id": "req_0000000000000005",
        "result": "pass",
        "reference": "本地证据",
        "summary": "通过",
        "reviewer": "甲"
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 8,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [
            {
              "id": "ecu",
              "name": "ECU",
              "kind": "entity",
              "entityType": "ecu",
              "children": [],
              "attributes": [],
              "x": 0.0,
              "y": 1.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            },
            {
              "id": "tester",
              "name": "终端",
              "kind": "entity",
              "entityType": "external",
              "children": [],
              "attributes": [],
              "x": 400.0,
              "y": 0.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            }
          ],
          "flows": [
            {
              "id": "can",
              "name": "消息",
              "source": "ecu",
              "target": "tester",
              "sourceSide": null,
              "targetSide": null,
              "labelOffset": {
                "x": 0,
                "y": 0
              },
              "protocol": "",
              "description": ""
            }
          ],
          "equal_spacing": true
        },
        "assets": [
          {
            "id": "asset_0000000000000002",
            "name": "控制指令",
            "description": "",
            "node_ids": [
              "ecu"
            ],
            "properties": [
              "integrity"
            ],
            "reference": null
          }
        ],
        "scenarios": [
          {
            "id": "scenario_0000000000000004",
            "title": "篡改",
            "asset_id": "asset_0000000000000002",
            "property": "integrity",
            "damage": "异常",
            "threat": "注入",
            "impact": {
              "safety": 3,
              "financial": 1,
              "operational": 2,
              "privacy": 0,
              "rationale": "影响依据"
            },
            "paths": [
              {
                "id": "path_0000000000000003",
                "name": "接入",
                "steps": "接入后发送",
                "feasibility": 3,
                "rationale": "测试",
                "reference": null
              }
            ],
            "references": []
          }
        ],
        "decisions": [
          {
            "scenario_id": "scenario_0000000000000004",
            "treatment": "reduce",
            "rationale": "分析",
            "owner": "负责人",
            "goal": "验证消息",
            "claim": "",
            "residual_impact": 2,
            "residual_feasibility": 1,
            "residual_rationale": "独立评估",
            "assessment_hash": "37ad7a31386fbbf3902d94cb4ae48f9cdb256bd732d5eea601094edcaa6ccb00",
            "decided_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "requirements": [
          {
            "id": "req_0000000000000005",
            "scenario_id": "scenario_0000000000000004",
            "statement": "验证来源",
            "allocation": "ECU",
            "verification": "篡改拒绝",
            "standard_refs": [],
            "reference": null
          }
        ],
        "evidence": [
          {
            "id": "evidence_0000000000000006",
            "requirement_id": "req_0000000000000005",
            "result": "pass",
            "reference": "本地证据",
            "summary": "通过",
            "reviewer": "甲",
            "requirement_hash": "5143d9e77b6b11aa9946fe4534a0c5bb142404fb9451092c3964efe1e1787980",
            "recorded_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": true,
        "baselines": [],
        "reference_imports": {},
        "regulation_refs": []
      },
      "entity_id": "evidence_0000000000000006",
      "review": {
        "ready": true,
        "issues": [],
        "evaluations": [
          {
            "scenario_id": "scenario_0000000000000004",
            "impact": 3,
            "feasibility": 3,
            "level": 5,
            "paths": [
              {
                "id": "path_0000000000000003",
                "level": 5
              }
            ],
            "decision_stale": false,
            "residual": 2
          }
        ],
        "requirements": [
          {
            "requirement_id": "req_0000000000000005",
            "status": "pass",
            "draft": false,
            "latest_evidence_id": "evidence_0000000000000006"
          }
        ],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": true
          },
          {
            "key": "assessment",
            "complete": true
          },
          {
            "key": "treatment",
            "complete": true
          },
          {
            "key": "verification",
            "complete": true
          }
        ]
      },
      "digest": "c9261ff6cf27553462c6868343bc1bd64fd0cf7886211de482be6a68fc7f9e80",
      "before_digest": "cf39899f8491785ee203c7f74eac912f85854b188016bd06b05951a4ca65d136"
    },
    {
      "action": "seal-baseline",
      "payload": {
        "name": "评审V1",
        "reviewer": "甲",
        "note": "完整评审"
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 9,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [
            {
              "id": "ecu",
              "name": "ECU",
              "kind": "entity",
              "entityType": "ecu",
              "children": [],
              "attributes": [],
              "x": 0.0,
              "y": 1.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            },
            {
              "id": "tester",
              "name": "终端",
              "kind": "entity",
              "entityType": "external",
              "children": [],
              "attributes": [],
              "x": 400.0,
              "y": 0.0,
              "width": 180,
              "height": 80,
              "description": "",
              "reference": null
            }
          ],
          "flows": [
            {
              "id": "can",
              "name": "消息",
              "source": "ecu",
              "target": "tester",
              "sourceSide": null,
              "targetSide": null,
              "labelOffset": {
                "x": 0,
                "y": 0
              },
              "protocol": "",
              "description": ""
            }
          ],
          "equal_spacing": true
        },
        "assets": [
          {
            "id": "asset_0000000000000002",
            "name": "控制指令",
            "description": "",
            "node_ids": [
              "ecu"
            ],
            "properties": [
              "integrity"
            ],
            "reference": null
          }
        ],
        "scenarios": [
          {
            "id": "scenario_0000000000000004",
            "title": "篡改",
            "asset_id": "asset_0000000000000002",
            "property": "integrity",
            "damage": "异常",
            "threat": "注入",
            "impact": {
              "safety": 3,
              "financial": 1,
              "operational": 2,
              "privacy": 0,
              "rationale": "影响依据"
            },
            "paths": [
              {
                "id": "path_0000000000000003",
                "name": "接入",
                "steps": "接入后发送",
                "feasibility": 3,
                "rationale": "测试",
                "reference": null
              }
            ],
            "references": []
          }
        ],
        "decisions": [
          {
            "scenario_id": "scenario_0000000000000004",
            "treatment": "reduce",
            "rationale": "分析",
            "owner": "负责人",
            "goal": "验证消息",
            "claim": "",
            "residual_impact": 2,
            "residual_feasibility": 1,
            "residual_rationale": "独立评估",
            "assessment_hash": "37ad7a31386fbbf3902d94cb4ae48f9cdb256bd732d5eea601094edcaa6ccb00",
            "decided_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "requirements": [
          {
            "id": "req_0000000000000005",
            "scenario_id": "scenario_0000000000000004",
            "statement": "验证来源",
            "allocation": "ECU",
            "verification": "篡改拒绝",
            "standard_refs": [],
            "reference": null
          }
        ],
        "evidence": [
          {
            "id": "evidence_0000000000000006",
            "requirement_id": "req_0000000000000005",
            "result": "pass",
            "reference": "本地证据",
            "summary": "通过",
            "reviewer": "甲",
            "requirement_hash": "5143d9e77b6b11aa9946fe4534a0c5bb142404fb9451092c3964efe1e1787980",
            "recorded_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": true,
        "baselines": [
          {
            "name": "评审V1",
            "reviewer": "甲",
            "note": "完整评审",
            "id": "baseline_0000000000000007",
            "revision": 8,
            "sealed_at": "2026-10-25T12:34:56.123456+00:00",
            "content_hash": "c9261ff6cf27553462c6868343bc1bd64fd0cf7886211de482be6a68fc7f9e80"
          }
        ],
        "reference_imports": {},
        "regulation_refs": []
      },
      "entity_id": "baseline_0000000000000007",
      "review": {
        "ready": true,
        "issues": [],
        "evaluations": [
          {
            "scenario_id": "scenario_0000000000000004",
            "impact": 3,
            "feasibility": 3,
            "level": 5,
            "paths": [
              {
                "id": "path_0000000000000003",
                "level": 5
              }
            ],
            "decision_stale": false,
            "residual": 2
          }
        ],
        "requirements": [
          {
            "requirement_id": "req_0000000000000005",
            "status": "pass",
            "draft": false,
            "latest_evidence_id": "evidence_0000000000000006"
          }
        ],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": true
          },
          {
            "key": "assessment",
            "complete": true
          },
          {
            "key": "treatment",
            "complete": true
          },
          {
            "key": "verification",
            "complete": true
          }
        ]
      },
      "digest": "ed991bc61f36d7bf399dd3f6bdc5dd0d39fcb8f64f4e00c294cf03301ea36143",
      "before_digest": "c9261ff6cf27553462c6868343bc1bd64fd0cf7886211de482be6a68fc7f9e80"
    },
    {
      "action": "replace-architecture",
      "payload": {
        "nodes": [
          {
            "id": "ecu",
            "name": "ECU",
            "kind": "entity",
            "entityType": "ecu",
            "children": [],
            "attributes": [],
            "x": 34,
            "y": 1.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          },
          {
            "id": "tester",
            "name": "终端",
            "kind": "entity",
            "entityType": "external",
            "children": [],
            "attributes": [],
            "x": 400.0,
            "y": 0.0,
            "width": 180,
            "height": 80,
            "description": "",
            "reference": null
          }
        ],
        "flows": [
          {
            "id": "can",
            "name": "消息",
            "source": "ecu",
            "target": "tester",
            "sourceSide": null,
            "targetSide": null,
            "labelOffset": {
              "x": 50.5,
              "y": -20
            },
            "protocol": "",
            "description": ""
          }
        ],
        "equal_spacing": true
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 10,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [
            {
              "id": "ecu",
              "name": "ECU",
              "kind": "entity",
              "entityType": "ecu",
              "children": [],
              "attributes": [],
              "x": 34.0,
              "y": 1.0,
              "width": 180.0,
              "height": 80.0,
              "description": "",
              "reference": null
            },
            {
              "id": "tester",
              "name": "终端",
              "kind": "entity",
              "entityType": "external",
              "children": [],
              "attributes": [],
              "x": 400.0,
              "y": 0.0,
              "width": 180.0,
              "height": 80.0,
              "description": "",
              "reference": null
            }
          ],
          "flows": [
            {
              "id": "can",
              "name": "消息",
              "source": "ecu",
              "target": "tester",
              "sourceSide": null,
              "targetSide": null,
              "labelOffset": {
                "x": 50.5,
                "y": -20.0
              },
              "protocol": "",
              "description": ""
            }
          ],
          "equal_spacing": true
        },
        "assets": [
          {
            "id": "asset_0000000000000002",
            "name": "控制指令",
            "description": "",
            "node_ids": [
              "ecu"
            ],
            "properties": [
              "integrity"
            ],
            "reference": null
          }
        ],
        "scenarios": [
          {
            "id": "scenario_0000000000000004",
            "title": "篡改",
            "asset_id": "asset_0000000000000002",
            "property": "integrity",
            "damage": "异常",
            "threat": "注入",
            "impact": {
              "safety": 3,
              "financial": 1,
              "operational": 2,
              "privacy": 0,
              "rationale": "影响依据"
            },
            "paths": [
              {
                "id": "path_0000000000000003",
                "name": "接入",
                "steps": "接入后发送",
                "feasibility": 3,
                "rationale": "测试",
                "reference": null
              }
            ],
            "references": []
          }
        ],
        "decisions": [
          {
            "scenario_id": "scenario_0000000000000004",
            "treatment": "reduce",
            "rationale": "分析",
            "owner": "负责人",
            "goal": "验证消息",
            "claim": "",
            "residual_impact": 2,
            "residual_feasibility": 1,
            "residual_rationale": "独立评估",
            "assessment_hash": "37ad7a31386fbbf3902d94cb4ae48f9cdb256bd732d5eea601094edcaa6ccb00",
            "decided_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "requirements": [
          {
            "id": "req_0000000000000005",
            "scenario_id": "scenario_0000000000000004",
            "statement": "验证来源",
            "allocation": "ECU",
            "verification": "篡改拒绝",
            "standard_refs": [],
            "reference": null
          }
        ],
        "evidence": [
          {
            "id": "evidence_0000000000000006",
            "requirement_id": "req_0000000000000005",
            "result": "pass",
            "reference": "本地证据",
            "summary": "通过",
            "reviewer": "甲",
            "requirement_hash": "5143d9e77b6b11aa9946fe4534a0c5bb142404fb9451092c3964efe1e1787980",
            "recorded_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": true,
        "baselines": [
          {
            "name": "评审V1",
            "reviewer": "甲",
            "note": "完整评审",
            "id": "baseline_0000000000000007",
            "revision": 8,
            "sealed_at": "2026-10-25T12:34:56.123456+00:00",
            "content_hash": "c9261ff6cf27553462c6868343bc1bd64fd0cf7886211de482be6a68fc7f9e80"
          }
        ],
        "reference_imports": {},
        "regulation_refs": []
      },
      "entity_id": "ws_0000000000000001",
      "review": {
        "ready": true,
        "issues": [],
        "evaluations": [
          {
            "scenario_id": "scenario_0000000000000004",
            "impact": 3,
            "feasibility": 3,
            "level": 5,
            "paths": [
              {
                "id": "path_0000000000000003",
                "level": 5
              }
            ],
            "decision_stale": false,
            "residual": 2
          }
        ],
        "requirements": [
          {
            "requirement_id": "req_0000000000000005",
            "status": "pass",
            "draft": false,
            "latest_evidence_id": "evidence_0000000000000006"
          }
        ],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": true
          },
          {
            "key": "assessment",
            "complete": true
          },
          {
            "key": "treatment",
            "complete": true
          },
          {
            "key": "verification",
            "complete": true
          }
        ]
      },
      "digest": "584d87ff6b6dc8139ad645cac69e45a17ee2705ea37556c75ed90a22cd1f93fd",
      "before_digest": "ed991bc61f36d7bf399dd3f6bdc5dd0d39fcb8f64f4e00c294cf03301ea36143"
    },
    {
      "action": "select-regulations",
      "payload": {
        "references": [
          "reg:test:1",
          "reg:test:1"
        ]
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 11,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [
            {
              "id": "ecu",
              "name": "ECU",
              "kind": "entity",
              "entityType": "ecu",
              "children": [],
              "attributes": [],
              "x": 34.0,
              "y": 1.0,
              "width": 180.0,
              "height": 80.0,
              "description": "",
              "reference": null
            },
            {
              "id": "tester",
              "name": "终端",
              "kind": "entity",
              "entityType": "external",
              "children": [],
              "attributes": [],
              "x": 400.0,
              "y": 0.0,
              "width": 180.0,
              "height": 80.0,
              "description": "",
              "reference": null
            }
          ],
          "flows": [
            {
              "id": "can",
              "name": "消息",
              "source": "ecu",
              "target": "tester",
              "sourceSide": null,
              "targetSide": null,
              "labelOffset": {
                "x": 50.5,
                "y": -20.0
              },
              "protocol": "",
              "description": ""
            }
          ],
          "equal_spacing": true
        },
        "assets": [
          {
            "id": "asset_0000000000000002",
            "name": "控制指令",
            "description": "",
            "node_ids": [
              "ecu"
            ],
            "properties": [
              "integrity"
            ],
            "reference": null
          }
        ],
        "scenarios": [
          {
            "id": "scenario_0000000000000004",
            "title": "篡改",
            "asset_id": "asset_0000000000000002",
            "property": "integrity",
            "damage": "异常",
            "threat": "注入",
            "impact": {
              "safety": 3,
              "financial": 1,
              "operational": 2,
              "privacy": 0,
              "rationale": "影响依据"
            },
            "paths": [
              {
                "id": "path_0000000000000003",
                "name": "接入",
                "steps": "接入后发送",
                "feasibility": 3,
                "rationale": "测试",
                "reference": null
              }
            ],
            "references": []
          }
        ],
        "decisions": [
          {
            "scenario_id": "scenario_0000000000000004",
            "treatment": "reduce",
            "rationale": "分析",
            "owner": "负责人",
            "goal": "验证消息",
            "claim": "",
            "residual_impact": 2,
            "residual_feasibility": 1,
            "residual_rationale": "独立评估",
            "assessment_hash": "37ad7a31386fbbf3902d94cb4ae48f9cdb256bd732d5eea601094edcaa6ccb00",
            "decided_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "requirements": [
          {
            "id": "req_0000000000000005",
            "scenario_id": "scenario_0000000000000004",
            "statement": "验证来源",
            "allocation": "ECU",
            "verification": "篡改拒绝",
            "standard_refs": [],
            "reference": null
          }
        ],
        "evidence": [
          {
            "id": "evidence_0000000000000006",
            "requirement_id": "req_0000000000000005",
            "result": "pass",
            "reference": "本地证据",
            "summary": "通过",
            "reviewer": "甲",
            "requirement_hash": "5143d9e77b6b11aa9946fe4534a0c5bb142404fb9451092c3964efe1e1787980",
            "recorded_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": true,
        "baselines": [
          {
            "name": "评审V1",
            "reviewer": "甲",
            "note": "完整评审",
            "id": "baseline_0000000000000007",
            "revision": 8,
            "sealed_at": "2026-10-25T12:34:56.123456+00:00",
            "content_hash": "c9261ff6cf27553462c6868343bc1bd64fd0cf7886211de482be6a68fc7f9e80"
          }
        ],
        "reference_imports": {},
        "regulation_refs": [
          "reg:test:1"
        ]
      },
      "entity_id": "ws_0000000000000001",
      "review": {
        "ready": true,
        "issues": [],
        "evaluations": [
          {
            "scenario_id": "scenario_0000000000000004",
            "impact": 3,
            "feasibility": 3,
            "level": 5,
            "paths": [
              {
                "id": "path_0000000000000003",
                "level": 5
              }
            ],
            "decision_stale": false,
            "residual": 2
          }
        ],
        "requirements": [
          {
            "requirement_id": "req_0000000000000005",
            "status": "pass",
            "draft": false,
            "latest_evidence_id": "evidence_0000000000000006"
          }
        ],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": true
          },
          {
            "key": "assessment",
            "complete": true
          },
          {
            "key": "treatment",
            "complete": true
          },
          {
            "key": "verification",
            "complete": true
          }
        ]
      },
      "digest": "a0190501b3745ae41c85935d4634f6469ab76c1942e63a141077de0d87069c31",
      "before_digest": "584d87ff6b6dc8139ad645cac69e45a17ee2705ea37556c75ed90a22cd1f93fd"
    },
    {
      "action": "specify-requirement",
      "payload": {
        "id": "req_0000000000000005",
        "scenario_id": "scenario_0000000000000004",
        "statement": "验证来源及计数器",
        "allocation": "ECU",
        "verification": "篡改拒绝",
        "standard_refs": [],
        "reference": null
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 12,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [
            {
              "id": "ecu",
              "name": "ECU",
              "kind": "entity",
              "entityType": "ecu",
              "children": [],
              "attributes": [],
              "x": 34.0,
              "y": 1.0,
              "width": 180.0,
              "height": 80.0,
              "description": "",
              "reference": null
            },
            {
              "id": "tester",
              "name": "终端",
              "kind": "entity",
              "entityType": "external",
              "children": [],
              "attributes": [],
              "x": 400.0,
              "y": 0.0,
              "width": 180.0,
              "height": 80.0,
              "description": "",
              "reference": null
            }
          ],
          "flows": [
            {
              "id": "can",
              "name": "消息",
              "source": "ecu",
              "target": "tester",
              "sourceSide": null,
              "targetSide": null,
              "labelOffset": {
                "x": 50.5,
                "y": -20.0
              },
              "protocol": "",
              "description": ""
            }
          ],
          "equal_spacing": true
        },
        "assets": [
          {
            "id": "asset_0000000000000002",
            "name": "控制指令",
            "description": "",
            "node_ids": [
              "ecu"
            ],
            "properties": [
              "integrity"
            ],
            "reference": null
          }
        ],
        "scenarios": [
          {
            "id": "scenario_0000000000000004",
            "title": "篡改",
            "asset_id": "asset_0000000000000002",
            "property": "integrity",
            "damage": "异常",
            "threat": "注入",
            "impact": {
              "safety": 3,
              "financial": 1,
              "operational": 2,
              "privacy": 0,
              "rationale": "影响依据"
            },
            "paths": [
              {
                "id": "path_0000000000000003",
                "name": "接入",
                "steps": "接入后发送",
                "feasibility": 3,
                "rationale": "测试",
                "reference": null
              }
            ],
            "references": []
          }
        ],
        "decisions": [
          {
            "scenario_id": "scenario_0000000000000004",
            "treatment": "reduce",
            "rationale": "分析",
            "owner": "负责人",
            "goal": "验证消息",
            "claim": "",
            "residual_impact": 2,
            "residual_feasibility": 1,
            "residual_rationale": "独立评估",
            "assessment_hash": "37ad7a31386fbbf3902d94cb4ae48f9cdb256bd732d5eea601094edcaa6ccb00",
            "decided_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "requirements": [
          {
            "id": "req_0000000000000005",
            "scenario_id": "scenario_0000000000000004",
            "statement": "验证来源及计数器",
            "allocation": "ECU",
            "verification": "篡改拒绝",
            "standard_refs": [],
            "reference": null
          }
        ],
        "evidence": [
          {
            "id": "evidence_0000000000000006",
            "requirement_id": "req_0000000000000005",
            "result": "pass",
            "reference": "本地证据",
            "summary": "通过",
            "reviewer": "甲",
            "requirement_hash": "5143d9e77b6b11aa9946fe4534a0c5bb142404fb9451092c3964efe1e1787980",
            "recorded_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": true,
        "baselines": [
          {
            "name": "评审V1",
            "reviewer": "甲",
            "note": "完整评审",
            "id": "baseline_0000000000000007",
            "revision": 8,
            "sealed_at": "2026-10-25T12:34:56.123456+00:00",
            "content_hash": "c9261ff6cf27553462c6868343bc1bd64fd0cf7886211de482be6a68fc7f9e80"
          }
        ],
        "reference_imports": {},
        "regulation_refs": [
          "reg:test:1"
        ]
      },
      "entity_id": "req_0000000000000005",
      "review": {
        "ready": false,
        "issues": [
          {
            "stage": "verification",
            "message": "安全需求缺少有效的通过验证记录",
            "reference": "req_0000000000000005"
          }
        ],
        "evaluations": [
          {
            "scenario_id": "scenario_0000000000000004",
            "impact": 3,
            "feasibility": 3,
            "level": 5,
            "paths": [
              {
                "id": "path_0000000000000003",
                "level": 5
              }
            ],
            "decision_stale": false,
            "residual": 2
          }
        ],
        "requirements": [
          {
            "requirement_id": "req_0000000000000005",
            "status": "stale",
            "draft": false,
            "latest_evidence_id": "evidence_0000000000000006"
          }
        ],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": true
          },
          {
            "key": "assessment",
            "complete": true
          },
          {
            "key": "treatment",
            "complete": true
          },
          {
            "key": "verification",
            "complete": false
          }
        ]
      },
      "digest": "98124234682f200c1239ecd16f0d2178457969fa013139adf347bc6e0a46d669",
      "before_digest": "a0190501b3745ae41c85935d4634f6469ab76c1942e63a141077de0d87069c31"
    },
    {
      "action": "record-evidence",
      "payload": {
        "requirement_id": "req_0000000000000005",
        "result": "fail",
        "reference": "失败证据",
        "summary": "失败",
        "reviewer": "乙"
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 13,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [
            {
              "id": "ecu",
              "name": "ECU",
              "kind": "entity",
              "entityType": "ecu",
              "children": [],
              "attributes": [],
              "x": 34.0,
              "y": 1.0,
              "width": 180.0,
              "height": 80.0,
              "description": "",
              "reference": null
            },
            {
              "id": "tester",
              "name": "终端",
              "kind": "entity",
              "entityType": "external",
              "children": [],
              "attributes": [],
              "x": 400.0,
              "y": 0.0,
              "width": 180.0,
              "height": 80.0,
              "description": "",
              "reference": null
            }
          ],
          "flows": [
            {
              "id": "can",
              "name": "消息",
              "source": "ecu",
              "target": "tester",
              "sourceSide": null,
              "targetSide": null,
              "labelOffset": {
                "x": 50.5,
                "y": -20.0
              },
              "protocol": "",
              "description": ""
            }
          ],
          "equal_spacing": true
        },
        "assets": [
          {
            "id": "asset_0000000000000002",
            "name": "控制指令",
            "description": "",
            "node_ids": [
              "ecu"
            ],
            "properties": [
              "integrity"
            ],
            "reference": null
          }
        ],
        "scenarios": [
          {
            "id": "scenario_0000000000000004",
            "title": "篡改",
            "asset_id": "asset_0000000000000002",
            "property": "integrity",
            "damage": "异常",
            "threat": "注入",
            "impact": {
              "safety": 3,
              "financial": 1,
              "operational": 2,
              "privacy": 0,
              "rationale": "影响依据"
            },
            "paths": [
              {
                "id": "path_0000000000000003",
                "name": "接入",
                "steps": "接入后发送",
                "feasibility": 3,
                "rationale": "测试",
                "reference": null
              }
            ],
            "references": []
          }
        ],
        "decisions": [
          {
            "scenario_id": "scenario_0000000000000004",
            "treatment": "reduce",
            "rationale": "分析",
            "owner": "负责人",
            "goal": "验证消息",
            "claim": "",
            "residual_impact": 2,
            "residual_feasibility": 1,
            "residual_rationale": "独立评估",
            "assessment_hash": "37ad7a31386fbbf3902d94cb4ae48f9cdb256bd732d5eea601094edcaa6ccb00",
            "decided_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "requirements": [
          {
            "id": "req_0000000000000005",
            "scenario_id": "scenario_0000000000000004",
            "statement": "验证来源及计数器",
            "allocation": "ECU",
            "verification": "篡改拒绝",
            "standard_refs": [],
            "reference": null
          }
        ],
        "evidence": [
          {
            "id": "evidence_0000000000000006",
            "requirement_id": "req_0000000000000005",
            "result": "pass",
            "reference": "本地证据",
            "summary": "通过",
            "reviewer": "甲",
            "requirement_hash": "5143d9e77b6b11aa9946fe4534a0c5bb142404fb9451092c3964efe1e1787980",
            "recorded_at": "2026-10-25T12:34:56.123456+00:00"
          },
          {
            "id": "evidence_0000000000000008",
            "requirement_id": "req_0000000000000005",
            "result": "fail",
            "reference": "失败证据",
            "summary": "失败",
            "reviewer": "乙",
            "requirement_hash": "1452362a7fd13ef76dd59bcb5005dd28047f521802d897167b45cb1270bd2fc3",
            "recorded_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": true,
        "baselines": [
          {
            "name": "评审V1",
            "reviewer": "甲",
            "note": "完整评审",
            "id": "baseline_0000000000000007",
            "revision": 8,
            "sealed_at": "2026-10-25T12:34:56.123456+00:00",
            "content_hash": "c9261ff6cf27553462c6868343bc1bd64fd0cf7886211de482be6a68fc7f9e80"
          }
        ],
        "reference_imports": {},
        "regulation_refs": [
          "reg:test:1"
        ]
      },
      "entity_id": "evidence_0000000000000008",
      "review": {
        "ready": false,
        "issues": [
          {
            "stage": "verification",
            "message": "安全需求缺少有效的通过验证记录",
            "reference": "req_0000000000000005"
          }
        ],
        "evaluations": [
          {
            "scenario_id": "scenario_0000000000000004",
            "impact": 3,
            "feasibility": 3,
            "level": 5,
            "paths": [
              {
                "id": "path_0000000000000003",
                "level": 5
              }
            ],
            "decision_stale": false,
            "residual": 2
          }
        ],
        "requirements": [
          {
            "requirement_id": "req_0000000000000005",
            "status": "fail",
            "draft": false,
            "latest_evidence_id": "evidence_0000000000000008"
          }
        ],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": true
          },
          {
            "key": "assessment",
            "complete": true
          },
          {
            "key": "treatment",
            "complete": true
          },
          {
            "key": "verification",
            "complete": false
          }
        ]
      },
      "digest": "c5cc807f654c63a226c98f189e8e3ea35bb5c94ba8109a3246961dc50d0b64fe",
      "before_digest": "98124234682f200c1239ecd16f0d2178457969fa013139adf347bc6e0a46d669"
    },
    {
      "action": "record-scenario",
      "payload": {
        "id": "scenario_0000000000000004",
        "title": "篡改",
        "asset_id": "asset_0000000000000002",
        "property": "integrity",
        "damage": "异常",
        "threat": "改变威胁",
        "impact": {
          "safety": 3,
          "financial": 1,
          "operational": 2,
          "privacy": 0,
          "rationale": "影响依据"
        },
        "paths": [
          {
            "id": "path_0000000000000003",
            "name": "接入",
            "steps": "接入后发送",
            "feasibility": 3,
            "rationale": "测试",
            "reference": null
          }
        ],
        "references": []
      },
      "expected": {
        "id": "ws_0000000000000001",
        "title": "Golden 项目",
        "description": "测试历史兼容",
        "revision": 14,
        "created_at": "2026-10-25T12:34:56.123456+00:00",
        "updated_at": "2026-10-25T12:34:56.123456+00:00",
        "item": {
          "purpose": "功能",
          "boundary": "边界",
          "environment": "环境",
          "assumptions": "假设",
          "owner": "负责人",
          "lifecycle": "概念阶段"
        },
        "architecture": {
          "nodes": [
            {
              "id": "ecu",
              "name": "ECU",
              "kind": "entity",
              "entityType": "ecu",
              "children": [],
              "attributes": [],
              "x": 34.0,
              "y": 1.0,
              "width": 180.0,
              "height": 80.0,
              "description": "",
              "reference": null
            },
            {
              "id": "tester",
              "name": "终端",
              "kind": "entity",
              "entityType": "external",
              "children": [],
              "attributes": [],
              "x": 400.0,
              "y": 0.0,
              "width": 180.0,
              "height": 80.0,
              "description": "",
              "reference": null
            }
          ],
          "flows": [
            {
              "id": "can",
              "name": "消息",
              "source": "ecu",
              "target": "tester",
              "sourceSide": null,
              "targetSide": null,
              "labelOffset": {
                "x": 50.5,
                "y": -20.0
              },
              "protocol": "",
              "description": ""
            }
          ],
          "equal_spacing": true
        },
        "assets": [
          {
            "id": "asset_0000000000000002",
            "name": "控制指令",
            "description": "",
            "node_ids": [
              "ecu"
            ],
            "properties": [
              "integrity"
            ],
            "reference": null
          }
        ],
        "scenarios": [
          {
            "id": "scenario_0000000000000004",
            "title": "篡改",
            "asset_id": "asset_0000000000000002",
            "property": "integrity",
            "damage": "异常",
            "threat": "改变威胁",
            "impact": {
              "safety": 3,
              "financial": 1,
              "operational": 2,
              "privacy": 0,
              "rationale": "影响依据"
            },
            "paths": [
              {
                "id": "path_0000000000000003",
                "name": "接入",
                "steps": "接入后发送",
                "feasibility": 3,
                "rationale": "测试",
                "reference": null
              }
            ],
            "references": []
          }
        ],
        "decisions": [
          {
            "scenario_id": "scenario_0000000000000004",
            "treatment": "reduce",
            "rationale": "分析",
            "owner": "负责人",
            "goal": "验证消息",
            "claim": "",
            "residual_impact": 2,
            "residual_feasibility": 1,
            "residual_rationale": "独立评估",
            "assessment_hash": "37ad7a31386fbbf3902d94cb4ae48f9cdb256bd732d5eea601094edcaa6ccb00",
            "decided_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "requirements": [
          {
            "id": "req_0000000000000005",
            "scenario_id": "scenario_0000000000000004",
            "statement": "验证来源及计数器",
            "allocation": "ECU",
            "verification": "篡改拒绝",
            "standard_refs": [],
            "reference": null
          }
        ],
        "evidence": [
          {
            "id": "evidence_0000000000000006",
            "requirement_id": "req_0000000000000005",
            "result": "pass",
            "reference": "本地证据",
            "summary": "通过",
            "reviewer": "甲",
            "requirement_hash": "5143d9e77b6b11aa9946fe4534a0c5bb142404fb9451092c3964efe1e1787980",
            "recorded_at": "2026-10-25T12:34:56.123456+00:00"
          },
          {
            "id": "evidence_0000000000000008",
            "requirement_id": "req_0000000000000005",
            "result": "fail",
            "reference": "失败证据",
            "summary": "失败",
            "reviewer": "乙",
            "requirement_hash": "1452362a7fd13ef76dd59bcb5005dd28047f521802d897167b45cb1270bd2fc3",
            "recorded_at": "2026-10-25T12:34:56.123456+00:00"
          }
        ],
        "method": {
          "name": "组织示例方法 · 4 × 4",
          "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
          "matrix": [
            [
              1,
              1,
              1,
              1
            ],
            [
              1,
              2,
              2,
              3
            ],
            [
              2,
              3,
              3,
              4
            ],
            [
              3,
              4,
              5,
              5
            ]
          ]
        },
        "method_confirmed": true,
        "baselines": [
          {
            "name": "评审V1",
            "reviewer": "甲",
            "note": "完整评审",
            "id": "baseline_0000000000000007",
            "revision": 8,
            "sealed_at": "2026-10-25T12:34:56.123456+00:00",
            "content_hash": "c9261ff6cf27553462c6868343bc1bd64fd0cf7886211de482be6a68fc7f9e80"
          }
        ],
        "reference_imports": {},
        "regulation_refs": [
          "reg:test:1"
        ]
      },
      "entity_id": "scenario_0000000000000004",
      "review": {
        "ready": false,
        "issues": [
          {
            "stage": "verification",
            "message": "安全需求缺少有效的通过验证记录",
            "reference": "req_0000000000000005"
          },
          {
            "stage": "treatment",
            "message": "评估依据已变化，需要重新确认处置决策",
            "reference": "scenario_0000000000000004"
          }
        ],
        "evaluations": [
          {
            "scenario_id": "scenario_0000000000000004",
            "impact": 3,
            "feasibility": 3,
            "level": 5,
            "paths": [
              {
                "id": "path_0000000000000003",
                "level": 5
              }
            ],
            "decision_stale": true,
            "residual": null
          }
        ],
        "requirements": [
          {
            "requirement_id": "req_0000000000000005",
            "status": "fail",
            "draft": false,
            "latest_evidence_id": "evidence_0000000000000008"
          }
        ],
        "stages": [
          {
            "key": "definition",
            "complete": true
          },
          {
            "key": "architecture",
            "complete": true
          },
          {
            "key": "assessment",
            "complete": true
          },
          {
            "key": "treatment",
            "complete": false
          },
          {
            "key": "verification",
            "complete": false
          }
        ]
      },
      "digest": "95bec22d5a3061ae4f9bc153f1a0bd64413fb101b5490a1a725c9fa273bdaaba",
      "before_digest": "c5cc807f654c63a226c98f189e8e3ea35bb5c94ba8109a3246961dc50d0b64fe"
    }
  ],
  "legacy": {
    "id": "ws_0000000000000001",
    "title": "Golden 项目",
    "description": "测试历史兼容",
    "revision": 0,
    "created_at": "2026-10-25T12:34:56.123456+00:00",
    "updated_at": "2026-10-25T12:34:56.123456+00:00",
    "item": {
      "purpose": "",
      "boundary": "",
      "environment": "",
      "assumptions": "",
      "owner": "",
      "lifecycle": "概念阶段"
    },
    "architecture": {
      "nodes": [
        {
          "id": "ecu",
          "name": "控制器",
          "kind": "process",
          "x": 0,
          "y": 0,
          "width": 180,
          "height": 80,
          "description": "",
          "reference": null
        },
        {
          "id": "store",
          "name": "存储",
          "kind": "store",
          "x": 50,
          "y": 0,
          "width": 180,
          "height": 80,
          "description": "密钥",
          "reference": "systems:1"
        },
        {
          "id": "tester",
          "name": "外部",
          "kind": "external",
          "x": 500,
          "y": 0,
          "width": 180,
          "height": 80,
          "description": "",
          "reference": null
        }
      ],
      "flows": [
        {
          "id": "f",
          "name": "读",
          "source": "store",
          "target": "tester",
          "protocol": "",
          "description": ""
        }
      ],
      "equal_spacing": true
    },
    "assets": [
      {
        "id": "a",
        "name": "密钥",
        "description": "",
        "node_ids": [
          "store",
          "ecu"
        ],
        "properties": [
          "integrity"
        ],
        "reference": null
      }
    ],
    "scenarios": [],
    "decisions": [],
    "requirements": [],
    "evidence": [],
    "method": {
      "name": "组织示例方法 · 4 × 4",
      "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
      "matrix": [
        [
          1,
          1,
          1,
          1
        ],
        [
          1,
          2,
          2,
          3
        ],
        [
          2,
          3,
          3,
          4
        ],
        [
          3,
          4,
          5,
          5
        ]
      ]
    },
    "method_confirmed": false,
    "baselines": [],
    "reference_imports": {},
    "regulation_refs": []
  },
  "migration_payload": {
    "nodes": [
      {
        "id": "ecu",
        "name": "控制器",
        "kind": "process",
        "x": 0,
        "y": 0,
        "width": 180,
        "height": 80,
        "description": "",
        "reference": null,
        "children": [
          {
            "id": "store",
            "name": "存储",
            "description": "密钥",
            "reference": "systems:1",
            "kind": "store"
          }
        ]
      },
      {
        "id": "tester",
        "name": "外部",
        "kind": "external",
        "x": 500,
        "y": 0,
        "width": 180,
        "height": 80,
        "description": "",
        "reference": null
      }
    ],
    "flows": [
      {
        "id": "f",
        "name": "读",
        "source": "store",
        "target": "tester",
        "protocol": "",
        "description": ""
      }
    ],
    "equal_spacing": true
  },
  "migrated": {
    "id": "ws_0000000000000001",
    "title": "Golden 项目",
    "description": "测试历史兼容",
    "revision": 1,
    "created_at": "2026-10-25T12:34:56.123456+00:00",
    "updated_at": "2026-10-25T12:34:56.123456+00:00",
    "item": {
      "purpose": "",
      "boundary": "",
      "environment": "",
      "assumptions": "",
      "owner": "",
      "lifecycle": "概念阶段"
    },
    "architecture": {
      "nodes": [
        {
          "id": "ecu",
          "name": "控制器",
          "kind": "entity",
          "entityType": "ecu",
          "children": [
            {
              "id": "store",
              "name": "存储",
              "kind": "store",
              "description": "密钥",
              "reference": "systems:1"
            }
          ],
          "attributes": [],
          "x": 0.0,
          "y": 0.0,
          "width": 180.0,
          "height": 80.0,
          "description": "",
          "reference": null
        },
        {
          "id": "tester",
          "name": "外部",
          "kind": "entity",
          "entityType": "external",
          "children": [],
          "attributes": [],
          "x": 500.0,
          "y": 0.0,
          "width": 180.0,
          "height": 80.0,
          "description": "",
          "reference": null
        }
      ],
      "flows": [
        {
          "id": "f",
          "name": "读",
          "source": "ecu",
          "target": "tester",
          "sourceSide": null,
          "targetSide": null,
          "labelOffset": {
            "x": 0,
            "y": 0
          },
          "protocol": "",
          "description": ""
        }
      ],
      "equal_spacing": true
    },
    "assets": [
      {
        "id": "a",
        "name": "密钥",
        "description": "",
        "node_ids": [
          "ecu"
        ],
        "properties": [
          "integrity"
        ],
        "reference": null
      }
    ],
    "scenarios": [],
    "decisions": [],
    "requirements": [],
    "evidence": [],
    "method": {
      "name": "组织示例方法 · 4 × 4",
      "rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
      "matrix": [
        [
          1,
          1,
          1,
          1
        ],
        [
          1,
          2,
          2,
          3
        ],
        [
          2,
          3,
          3,
          4
        ],
        [
          3,
          4,
          5,
          5
        ]
      ]
    },
    "method_confirmed": false,
    "baselines": [],
    "reference_imports": {},
    "regulation_refs": []
  },
  "migration_review": {
    "ready": false,
    "issues": [
      {
        "stage": "definition",
        "message": "补全功能、边界、运行环境、假设和责任人",
        "reference": null
      },
      {
        "stage": "assessment",
        "message": "确认组织采用的风险评估方法",
        "reference": null
      },
      {
        "stage": "assessment",
        "message": "资产“密钥”的 integrity 尚无风险场景",
        "reference": "a"
      },
      {
        "stage": "assessment",
        "message": "尚未建立损害场景、威胁及攻击路径",
        "reference": null
      }
    ],
    "evaluations": [],
    "requirements": [],
    "stages": [
      {
        "key": "definition",
        "complete": false
      },
      {
        "key": "architecture",
        "complete": false
      },
      {
        "key": "assessment",
        "complete": false
      },
      {
        "key": "treatment",
        "complete": false
      },
      {
        "key": "verification",
        "complete": false
      }
    ]
  }
}`
