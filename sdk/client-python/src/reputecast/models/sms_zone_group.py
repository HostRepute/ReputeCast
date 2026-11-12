from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="SmsZoneGroup")



@_attrs_define
class SmsZoneGroup:
    """ 
        Attributes:
            countries (list[str]):
            from_ (datetime.datetime):
            held (bool):
            local_from (str):
            local_to (str):
            recipients (int):
            to (datetime.datetime):
            until (datetime.datetime):
     """

    countries: list[str]
    from_: datetime.datetime
    held: bool
    local_from: str
    local_to: str
    recipients: int
    to: datetime.datetime
    until: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        countries = self.countries



        from_ = self.from_.isoformat()

        held = self.held

        local_from = self.local_from

        local_to = self.local_to

        recipients = self.recipients

        to = self.to.isoformat()

        until = self.until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "countries": countries,
            "from": from_,
            "held": held,
            "localFrom": local_from,
            "localTo": local_to,
            "recipients": recipients,
            "to": to,
            "until": until,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        countries = cast(list[str], d.pop("countries"))


        from_ = datetime.datetime.fromisoformat(d.pop("from"))




        held = d.pop("held")

        local_from = d.pop("localFrom")

        local_to = d.pop("localTo")

        recipients = d.pop("recipients")

        to = datetime.datetime.fromisoformat(d.pop("to"))




        until = datetime.datetime.fromisoformat(d.pop("until"))




        sms_zone_group = cls(
            countries=countries,
            from_=from_,
            held=held,
            local_from=local_from,
            local_to=local_to,
            recipients=recipients,
            to=to,
            until=until,
        )

        return sms_zone_group

