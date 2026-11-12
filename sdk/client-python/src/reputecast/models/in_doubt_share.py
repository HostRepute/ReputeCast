from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="InDoubtShare")



@_attrs_define
class InDoubtShare:
    """ 
        Attributes:
            count (int):
            idempotent (bool):
            provider (str):
            provider_id (str):
            safe (bool):
            short (str):
            since (datetime.datetime):
            safe_until (datetime.datetime | Unset):
            window_seconds (int | Unset):
     """

    count: int
    idempotent: bool
    provider: str
    provider_id: str
    safe: bool
    short: str
    since: datetime.datetime
    safe_until: datetime.datetime | Unset = UNSET
    window_seconds: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        idempotent = self.idempotent

        provider = self.provider

        provider_id = self.provider_id

        safe = self.safe

        short = self.short

        since = self.since.isoformat()

        safe_until: str | Unset = UNSET
        if not isinstance(self.safe_until, Unset):
            safe_until = self.safe_until.isoformat()

        window_seconds = self.window_seconds


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "idempotent": idempotent,
            "provider": provider,
            "providerId": provider_id,
            "safe": safe,
            "short": short,
            "since": since,
        })
        if safe_until is not UNSET:
            field_dict["safeUntil"] = safe_until
        if window_seconds is not UNSET:
            field_dict["windowSeconds"] = window_seconds

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        idempotent = d.pop("idempotent")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        safe = d.pop("safe")

        short = d.pop("short")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        _safe_until = d.pop("safeUntil", UNSET)
        safe_until: datetime.datetime | Unset
        if isinstance(_safe_until,  Unset):
            safe_until = UNSET
        else:
            safe_until = datetime.datetime.fromisoformat(_safe_until)




        window_seconds = d.pop("windowSeconds", UNSET)

        in_doubt_share = cls(
            count=count,
            idempotent=idempotent,
            provider=provider,
            provider_id=provider_id,
            safe=safe,
            short=short,
            since=since,
            safe_until=safe_until,
            window_seconds=window_seconds,
        )

        return in_doubt_share

