from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_node_type import AutomationNodeType
from typing import cast

if TYPE_CHECKING:
  from ..models.automation_node_data import AutomationNodeData
  from ..models.automation_node_position import AutomationNodePosition





T = TypeVar("T", bound="AutomationNode")



@_attrs_define
class AutomationNode:
    """ 
        Attributes:
            data (AutomationNodeData):
            id (str):
            position (AutomationNodePosition):
            type_ (AutomationNodeType):
     """

    data: AutomationNodeData
    id: str
    position: AutomationNodePosition
    type_: AutomationNodeType





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_node_data import AutomationNodeData # noqa: PLC0415
        from ..models.automation_node_position import AutomationNodePosition # noqa: PLC0415
        data = self.data.to_dict()

        id = self.id

        position = self.position.to_dict()

        type_ = self.type_.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "data": data,
            "id": id,
            "position": position,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_node_data import AutomationNodeData # noqa: PLC0415
        from ..models.automation_node_position import AutomationNodePosition # noqa: PLC0415
        d = dict(src_dict)
        data = AutomationNodeData.from_dict(d.pop("data"))




        id = d.pop("id")

        position = AutomationNodePosition.from_dict(d.pop("position"))




        type_ = AutomationNodeType(d.pop("type"))




        automation_node = cls(
            data=data,
            id=id,
            position=position,
            type_=type_,
        )

        return automation_node

