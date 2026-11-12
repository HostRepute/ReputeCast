from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.segment_group_match import SegmentGroupMatch
from typing import cast

if TYPE_CHECKING:
  from ..models.segment_condition import SegmentCondition





T = TypeVar("T", bound="SegmentGroup")



@_attrs_define
class SegmentGroup:
    """ 
        Attributes:
            conditions (list[SegmentCondition]):
            match (SegmentGroupMatch): all = every condition, any = at least one
     """

    conditions: list[SegmentCondition]
    match: SegmentGroupMatch





    def to_dict(self) -> dict[str, Any]:
        from ..models.segment_condition import SegmentCondition # noqa: PLC0415
        conditions = []
        for conditions_item_data in self.conditions:
            conditions_item = conditions_item_data.to_dict()
            conditions.append(conditions_item)



        match = self.match.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "conditions": conditions,
            "match": match,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.segment_condition import SegmentCondition # noqa: PLC0415
        d = dict(src_dict)
        conditions = []
        _conditions = d.pop("conditions")
        for conditions_item_data in (_conditions):
            conditions_item = SegmentCondition.from_dict(conditions_item_data)



            conditions.append(conditions_item)


        match = SegmentGroupMatch(d.pop("match"))




        segment_group = cls(
            conditions=conditions,
            match=match,
        )

        return segment_group

