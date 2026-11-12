from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsCheckSender")



@_attrs_define
class SmsCheckSender:
    """ 
        Attributes:
            countries (list[str]):
            registration (str):
            sender (str):
            since (datetime.datetime | Unset):
     """

    countries: list[str]
    registration: str
    sender: str
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        countries = self.countries



        registration = self.registration

        sender = self.sender

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "countries": countries,
            "registration": registration,
            "sender": sender,
        })
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        countries = cast(list[str], d.pop("countries"))


        registration = d.pop("registration")

        sender = d.pop("sender")

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        sms_check_sender = cls(
            countries=countries,
            registration=registration,
            sender=sender,
            since=since,
        )

        return sms_check_sender

