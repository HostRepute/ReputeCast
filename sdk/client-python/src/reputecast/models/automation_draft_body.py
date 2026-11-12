from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.automation_graph import AutomationGraph





T = TypeVar("T", bound="AutomationDraftBody")



@_attrs_define
class AutomationDraftBody:
    """ 
        Attributes:
            graph (AutomationGraph):
            revision (int):
     """

    graph: AutomationGraph
    revision: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_graph import AutomationGraph # noqa: PLC0415
        graph = self.graph.to_dict()

        revision = self.revision


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "graph": graph,
            "revision": revision,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_graph import AutomationGraph # noqa: PLC0415
        d = dict(src_dict)
        graph = AutomationGraph.from_dict(d.pop("graph"))




        revision = d.pop("revision")

        automation_draft_body = cls(
            graph=graph,
            revision=revision,
        )

        return automation_draft_body

