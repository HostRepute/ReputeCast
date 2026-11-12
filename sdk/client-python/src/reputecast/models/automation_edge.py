from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.automation_edge_data import AutomationEdgeData





T = TypeVar("T", bound="AutomationEdge")



@_attrs_define
class AutomationEdge:
    """ 
        Attributes:
            data (AutomationEdgeData):
            id (str):
            source (str):
            target (str):
     """

    data: AutomationEdgeData
    id: str
    source: str
    target: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_edge_data import AutomationEdgeData # noqa: PLC0415
        data = self.data.to_dict()

        id = self.id

        source = self.source

        target = self.target


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "data": data,
            "id": id,
            "source": source,
            "target": target,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_edge_data import AutomationEdgeData # noqa: PLC0415
        d = dict(src_dict)
        data = AutomationEdgeData.from_dict(d.pop("data"))




        id = d.pop("id")

        source = d.pop("source")

        target = d.pop("target")

        automation_edge = cls(
            data=data,
            id=id,
            source=source,
            target=target,
        )

        return automation_edge

