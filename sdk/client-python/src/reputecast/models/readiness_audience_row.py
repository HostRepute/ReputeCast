from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_audience_row_action import ReadinessAudienceRowAction
from ..models.readiness_audience_row_kind import ReadinessAudienceRowKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="ReadinessAudienceRow")



@_attrs_define
class ReadinessAudienceRow:
    """ 
        Attributes:
            count (int):
            kind (ReadinessAudienceRowKind):
            label (str):
            action (ReadinessAudienceRowAction | Unset):
     """

    count: int
    kind: ReadinessAudienceRowKind
    label: str
    action: ReadinessAudienceRowAction | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        kind = self.kind.value

        label = self.label

        action: str | Unset = UNSET
        if not isinstance(self.action, Unset):
            action = self.action.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "kind": kind,
            "label": label,
        })
        if action is not UNSET:
            field_dict["action"] = action

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        kind = ReadinessAudienceRowKind(d.pop("kind"))




        label = d.pop("label")

        _action = d.pop("action", UNSET)
        action: ReadinessAudienceRowAction | Unset
        if isinstance(_action,  Unset):
            action = UNSET
        else:
            action = ReadinessAudienceRowAction(_action)




        readiness_audience_row = cls(
            count=count,
            kind=kind,
            label=label,
            action=action,
        )

        return readiness_audience_row

