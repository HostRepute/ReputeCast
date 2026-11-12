from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_profile_preferences_frequency import ContactProfilePreferencesFrequency
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ContactProfilePreferences")



@_attrs_define
class ContactProfilePreferences:
    """ 
        Attributes:
            fewer_emails (bool):
            frequency (ContactProfilePreferencesFrequency):
            paused_until (datetime.datetime | Unset):
     """

    fewer_emails: bool
    frequency: ContactProfilePreferencesFrequency
    paused_until: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        fewer_emails = self.fewer_emails

        frequency = self.frequency.value

        paused_until: str | Unset = UNSET
        if not isinstance(self.paused_until, Unset):
            paused_until = self.paused_until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fewerEmails": fewer_emails,
            "frequency": frequency,
        })
        if paused_until is not UNSET:
            field_dict["pausedUntil"] = paused_until

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fewer_emails = d.pop("fewerEmails")

        frequency = ContactProfilePreferencesFrequency(d.pop("frequency"))




        _paused_until = d.pop("pausedUntil", UNSET)
        paused_until: datetime.datetime | Unset
        if isinstance(_paused_until,  Unset):
            paused_until = UNSET
        else:
            paused_until = datetime.datetime.fromisoformat(_paused_until)




        contact_profile_preferences = cls(
            fewer_emails=fewer_emails,
            frequency=frequency,
            paused_until=paused_until,
        )

        return contact_profile_preferences

