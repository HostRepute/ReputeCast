from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_zone_step_status import SmsZoneStepStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsZoneStep")



@_attrs_define
class SmsZoneStep:
    """ 
        Attributes:
            abbr (str):
            at (datetime.datetime):
            countries (list[str]):
            delivered (int):
            local (str):
            sent (int):
            status (SmsZoneStepStatus):
            total (int):
            zone (str):
            region (str | Unset):
     """

    abbr: str
    at: datetime.datetime
    countries: list[str]
    delivered: int
    local: str
    sent: int
    status: SmsZoneStepStatus
    total: int
    zone: str
    region: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        abbr = self.abbr

        at = self.at.isoformat()

        countries = self.countries



        delivered = self.delivered

        local = self.local

        sent = self.sent

        status = self.status.value

        total = self.total

        zone = self.zone

        region = self.region


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "abbr": abbr,
            "at": at,
            "countries": countries,
            "delivered": delivered,
            "local": local,
            "sent": sent,
            "status": status,
            "total": total,
            "zone": zone,
        })
        if region is not UNSET:
            field_dict["region"] = region

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        abbr = d.pop("abbr")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        countries = cast(list[str], d.pop("countries"))


        delivered = d.pop("delivered")

        local = d.pop("local")

        sent = d.pop("sent")

        status = SmsZoneStepStatus(d.pop("status"))




        total = d.pop("total")

        zone = d.pop("zone")

        region = d.pop("region", UNSET)

        sms_zone_step = cls(
            abbr=abbr,
            at=at,
            countries=countries,
            delivered=delivered,
            local=local,
            sent=sent,
            status=status,
            total=total,
            zone=zone,
            region=region,
        )

        return sms_zone_step

