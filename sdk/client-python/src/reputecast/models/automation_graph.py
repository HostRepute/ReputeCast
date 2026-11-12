from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.automation_edge import AutomationEdge
  from ..models.automation_node import AutomationNode





T = TypeVar("T", bound="AutomationGraph")



@_attrs_define
class AutomationGraph:
    """ 
        Attributes:
            edges (list[AutomationEdge]):
            nodes (list[AutomationNode]):
     """

    edges: list[AutomationEdge]
    nodes: list[AutomationNode]





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_edge import AutomationEdge # noqa: PLC0415
        from ..models.automation_node import AutomationNode # noqa: PLC0415
        edges = []
        for edges_item_data in self.edges:
            edges_item = edges_item_data.to_dict()
            edges.append(edges_item)



        nodes = []
        for nodes_item_data in self.nodes:
            nodes_item = nodes_item_data.to_dict()
            nodes.append(nodes_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "edges": edges,
            "nodes": nodes,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_edge import AutomationEdge # noqa: PLC0415
        from ..models.automation_node import AutomationNode # noqa: PLC0415
        d = dict(src_dict)
        edges = []
        _edges = d.pop("edges")
        for edges_item_data in (_edges):
            edges_item = AutomationEdge.from_dict(edges_item_data)



            edges.append(edges_item)


        nodes = []
        _nodes = d.pop("nodes")
        for nodes_item_data in (_nodes):
            nodes_item = AutomationNode.from_dict(nodes_item_data)



            nodes.append(nodes_item)


        automation_graph = cls(
            edges=edges,
            nodes=nodes,
        )

        return automation_graph

