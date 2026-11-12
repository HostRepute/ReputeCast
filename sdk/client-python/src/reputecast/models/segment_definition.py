from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.segment_group import SegmentGroup





T = TypeVar("T", bound="SegmentDefinition")



@_attrs_define
class SegmentDefinition:
    """ 
        Attributes:
            groups (list[SegmentGroup]):
     """

    groups: list[SegmentGroup]





    def to_dict(self) -> dict[str, Any]:
        from ..models.segment_group import SegmentGroup # noqa: PLC0415
        groups = []
        for groups_item_data in self.groups:
            groups_item = groups_item_data.to_dict()
            groups.append(groups_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "groups": groups,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.segment_group import SegmentGroup # noqa: PLC0415
        d = dict(src_dict)
        groups = []
        _groups = d.pop("groups")
        for groups_item_data in (_groups):
            groups_item = SegmentGroup.from_dict(groups_item_data)



            groups.append(groups_item)


        segment_definition = cls(
            groups=groups,
        )

        return segment_definition

