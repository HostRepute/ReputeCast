from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ApprovalCan")



@_attrs_define
class ApprovalCan:
    """ 
        Attributes:
            cancel (bool):
            comment (bool):
            decide (bool):
            is_author (bool):
            remind (bool):
            restore (bool):
            schedule (bool):
            decide_reason (str | Unset):
     """

    cancel: bool
    comment: bool
    decide: bool
    is_author: bool
    remind: bool
    restore: bool
    schedule: bool
    decide_reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        cancel = self.cancel

        comment = self.comment

        decide = self.decide

        is_author = self.is_author

        remind = self.remind

        restore = self.restore

        schedule = self.schedule

        decide_reason = self.decide_reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cancel": cancel,
            "comment": comment,
            "decide": decide,
            "isAuthor": is_author,
            "remind": remind,
            "restore": restore,
            "schedule": schedule,
        })
        if decide_reason is not UNSET:
            field_dict["decideReason"] = decide_reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cancel = d.pop("cancel")

        comment = d.pop("comment")

        decide = d.pop("decide")

        is_author = d.pop("isAuthor")

        remind = d.pop("remind")

        restore = d.pop("restore")

        schedule = d.pop("schedule")

        decide_reason = d.pop("decideReason", UNSET)

        approval_can = cls(
            cancel=cancel,
            comment=comment,
            decide=decide,
            is_author=is_author,
            remind=remind,
            restore=restore,
            schedule=schedule,
            decide_reason=decide_reason,
        )

        return approval_can

