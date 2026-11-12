from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_dismissal_reason import ReadinessDismissalReason
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReadinessDismissal")



@_attrs_define
class ReadinessDismissal:
    """ 
        Attributes:
            at (datetime.datetime):
            by (str):
            count (int):
            id (str):
            reason (ReadinessDismissalReason):
            reason_text (str):
            note (str | Unset):
     """

    at: datetime.datetime
    by: str
    count: int
    id: str
    reason: ReadinessDismissalReason
    reason_text: str
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        by = self.by

        count = self.count

        id = self.id

        reason = self.reason.value

        reason_text = self.reason_text

        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "by": by,
            "count": count,
            "id": id,
            "reason": reason,
            "reasonText": reason_text,
        })
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        by = d.pop("by")

        count = d.pop("count")

        id = d.pop("id")

        reason = ReadinessDismissalReason(d.pop("reason"))




        reason_text = d.pop("reasonText")

        note = d.pop("note", UNSET)

        readiness_dismissal = cls(
            at=at,
            by=by,
            count=count,
            id=id,
            reason=reason,
            reason_text=reason_text,
            note=note,
        )

        return readiness_dismissal

