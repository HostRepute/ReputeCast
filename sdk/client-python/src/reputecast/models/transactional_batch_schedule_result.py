from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalBatchScheduleResult")



@_attrs_define
class TransactionalBatchScheduleResult:
    """ 
        Attributes:
            changed (int):
            matched (int):
            skipped (int):
            send_at (datetime.datetime | Unset):
     """

    changed: int
    matched: int
    skipped: int
    send_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        changed = self.changed

        matched = self.matched

        skipped = self.skipped

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changed": changed,
            "matched": matched,
            "skipped": skipped,
        })
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        changed = d.pop("changed")

        matched = d.pop("matched")

        skipped = d.pop("skipped")

        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        transactional_batch_schedule_result = cls(
            changed=changed,
            matched=matched,
            skipped=skipped,
            send_at=send_at,
        )

        return transactional_batch_schedule_result

