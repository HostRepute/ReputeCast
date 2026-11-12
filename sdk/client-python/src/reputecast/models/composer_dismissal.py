from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_dismissal_reason import ComposerDismissalReason
from typing import cast
import datetime






T = TypeVar("T", bound="ComposerDismissal")



@_attrs_define
class ComposerDismissal:
    """ 
        Attributes:
            at (datetime.datetime):
            by (str):
            check_id (str):
            id (str):
            key (str):
            note (str):
            reason (ComposerDismissalReason):
            reason_text (str):
     """

    at: datetime.datetime
    by: str
    check_id: str
    id: str
    key: str
    note: str
    reason: ComposerDismissalReason
    reason_text: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        by = self.by

        check_id = self.check_id

        id = self.id

        key = self.key

        note = self.note

        reason = self.reason.value

        reason_text = self.reason_text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "by": by,
            "checkId": check_id,
            "id": id,
            "key": key,
            "note": note,
            "reason": reason,
            "reasonText": reason_text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        by = d.pop("by")

        check_id = d.pop("checkId")

        id = d.pop("id")

        key = d.pop("key")

        note = d.pop("note")

        reason = ComposerDismissalReason(d.pop("reason"))




        reason_text = d.pop("reasonText")

        composer_dismissal = cls(
            at=at,
            by=by,
            check_id=check_id,
            id=id,
            key=key,
            note=note,
            reason=reason,
            reason_text=reason_text,
        )

        return composer_dismissal

