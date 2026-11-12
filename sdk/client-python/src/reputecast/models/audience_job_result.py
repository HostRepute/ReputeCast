from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AudienceJobResult")



@_attrs_define
class AudienceJobResult:
    """ 
        Attributes:
            confirmed (int | Unset): Skipped: confirmed already
            cooldown (int | Unset): Skipped: got one in the last 10 minutes, or five today
            failed (int | Unset): The System provider refused the email
            rows (int | Unset):
            sent (int | Unset):
            suppressed (int | Unset): Skipped: on the suppression list
     """

    confirmed: int | Unset = UNSET
    cooldown: int | Unset = UNSET
    failed: int | Unset = UNSET
    rows: int | Unset = UNSET
    sent: int | Unset = UNSET
    suppressed: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        confirmed = self.confirmed

        cooldown = self.cooldown

        failed = self.failed

        rows = self.rows

        sent = self.sent

        suppressed = self.suppressed


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if confirmed is not UNSET:
            field_dict["confirmed"] = confirmed
        if cooldown is not UNSET:
            field_dict["cooldown"] = cooldown
        if failed is not UNSET:
            field_dict["failed"] = failed
        if rows is not UNSET:
            field_dict["rows"] = rows
        if sent is not UNSET:
            field_dict["sent"] = sent
        if suppressed is not UNSET:
            field_dict["suppressed"] = suppressed

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        confirmed = d.pop("confirmed", UNSET)

        cooldown = d.pop("cooldown", UNSET)

        failed = d.pop("failed", UNSET)

        rows = d.pop("rows", UNSET)

        sent = d.pop("sent", UNSET)

        suppressed = d.pop("suppressed", UNSET)

        audience_job_result = cls(
            confirmed=confirmed,
            cooldown=cooldown,
            failed=failed,
            rows=rows,
            sent=sent,
            suppressed=suppressed,
        )

        return audience_job_result

