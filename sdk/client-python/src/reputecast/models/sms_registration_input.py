from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_registration_input_kind import SmsRegistrationInputKind
from ..models.sms_registration_input_status import SmsRegistrationInputStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsRegistrationInput")



@_attrs_define
class SmsRegistrationInput:
    """ 
        Attributes:
            kind (SmsRegistrationInputKind):
            on (datetime.datetime): The day the provider gave this status.
            status (SmsRegistrationInputStatus):
            country (str | Unset):
            name (str | Unset):
            reason (str | Unset):
     """

    kind: SmsRegistrationInputKind
    on: datetime.datetime
    status: SmsRegistrationInputStatus
    country: str | Unset = UNSET
    name: str | Unset = UNSET
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        on = self.on.isoformat()

        status = self.status.value

        country = self.country

        name = self.name

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "on": on,
            "status": status,
        })
        if country is not UNSET:
            field_dict["country"] = country
        if name is not UNSET:
            field_dict["name"] = name
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = SmsRegistrationInputKind(d.pop("kind"))




        on = datetime.datetime.fromisoformat(d.pop("on"))




        status = SmsRegistrationInputStatus(d.pop("status"))




        country = d.pop("country", UNSET)

        name = d.pop("name", UNSET)

        reason = d.pop("reason", UNSET)

        sms_registration_input = cls(
            kind=kind,
            on=on,
            status=status,
            country=country,
            name=name,
            reason=reason,
        )

        return sms_registration_input

