from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.approval_change_field import ApprovalChangeField
from ..types import UNSET, Unset






T = TypeVar("T", bound="ApprovalChange")



@_attrs_define
class ApprovalChange:
    """ 
        Attributes:
            after (str):
            before (str):
            field (ApprovalChangeField):
            label (str):
            what (str | Unset):
     """

    after: str
    before: str
    field: ApprovalChangeField
    label: str
    what: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        after = self.after

        before = self.before

        field = self.field.value

        label = self.label

        what = self.what


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "after": after,
            "before": before,
            "field": field,
            "label": label,
        })
        if what is not UNSET:
            field_dict["what"] = what

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        after = d.pop("after")

        before = d.pop("before")

        field = ApprovalChangeField(d.pop("field"))




        label = d.pop("label")

        what = d.pop("what", UNSET)

        approval_change = cls(
            after=after,
            before=before,
            field=field,
            label=label,
            what=what,
        )

        return approval_change

