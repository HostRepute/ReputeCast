from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_opt_out_sync_row_mode import SmsOptOutSyncRowMode
from ..models.sms_opt_out_sync_row_status import SmsOptOutSyncRowStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsOptOutSyncRow")



@_attrs_define
class SmsOptOutSyncRow:
    """ 
        Attributes:
            adapter (str):
            mode (SmsOptOutSyncRowMode):
            provider (str):
            pulled (int):
            pushed (int):
            status (SmsOptOutSyncRowStatus):
            detail (str | Unset):
            last_sync_at (datetime.datetime | Unset):
     """

    adapter: str
    mode: SmsOptOutSyncRowMode
    provider: str
    pulled: int
    pushed: int
    status: SmsOptOutSyncRowStatus
    detail: str | Unset = UNSET
    last_sync_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        mode = self.mode.value

        provider = self.provider

        pulled = self.pulled

        pushed = self.pushed

        status = self.status.value

        detail = self.detail

        last_sync_at: str | Unset = UNSET
        if not isinstance(self.last_sync_at, Unset):
            last_sync_at = self.last_sync_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "mode": mode,
            "provider": provider,
            "pulled": pulled,
            "pushed": pushed,
            "status": status,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail
        if last_sync_at is not UNSET:
            field_dict["lastSyncAt"] = last_sync_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        mode = SmsOptOutSyncRowMode(d.pop("mode"))




        provider = d.pop("provider")

        pulled = d.pop("pulled")

        pushed = d.pop("pushed")

        status = SmsOptOutSyncRowStatus(d.pop("status"))




        detail = d.pop("detail", UNSET)

        _last_sync_at = d.pop("lastSyncAt", UNSET)
        last_sync_at: datetime.datetime | Unset
        if isinstance(_last_sync_at,  Unset):
            last_sync_at = UNSET
        else:
            last_sync_at = datetime.datetime.fromisoformat(_last_sync_at)




        sms_opt_out_sync_row = cls(
            adapter=adapter,
            mode=mode,
            provider=provider,
            pulled=pulled,
            pushed=pushed,
            status=status,
            detail=detail,
            last_sync_at=last_sync_at,
        )

        return sms_opt_out_sync_row

