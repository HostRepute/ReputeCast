from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.public_turnstile_status import PublicTurnstileStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.public_turnstile_report import PublicTurnstileReport





T = TypeVar("T", bound="PublicTurnstile")



@_attrs_define
class PublicTurnstile:
    """ 
        Attributes:
            error (str): Why Cloudflare rejected the keys, with its error code
            has_secret (bool):
            site_key (str):
            status (PublicTurnstileStatus):
            checked_at (datetime.datetime | Unset):
            reported (PublicTurnstileReport | Unset):
     """

    error: str
    has_secret: bool
    site_key: str
    status: PublicTurnstileStatus
    checked_at: datetime.datetime | Unset = UNSET
    reported: PublicTurnstileReport | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.public_turnstile_report import PublicTurnstileReport # noqa: PLC0415
        error = self.error

        has_secret = self.has_secret

        site_key = self.site_key

        status = self.status.value

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        reported: dict[str, Any] | Unset = UNSET
        if not isinstance(self.reported, Unset):
            reported = self.reported.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "error": error,
            "hasSecret": has_secret,
            "siteKey": site_key,
            "status": status,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if reported is not UNSET:
            field_dict["reported"] = reported

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.public_turnstile_report import PublicTurnstileReport # noqa: PLC0415
        d = dict(src_dict)
        error = d.pop("error")

        has_secret = d.pop("hasSecret")

        site_key = d.pop("siteKey")

        status = PublicTurnstileStatus(d.pop("status"))




        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        _reported = d.pop("reported", UNSET)
        reported: PublicTurnstileReport | Unset
        if isinstance(_reported,  Unset):
            reported = UNSET
        else:
            reported = PublicTurnstileReport.from_dict(_reported)




        public_turnstile = cls(
            error=error,
            has_secret=has_secret,
            site_key=site_key,
            status=status,
            checked_at=checked_at,
            reported=reported,
        )

        return public_turnstile

