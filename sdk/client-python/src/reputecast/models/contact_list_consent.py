from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_list_consent_status import ContactListConsentStatus
from typing import cast

if TYPE_CHECKING:
  from ..models.contact_consent_event import ContactConsentEvent





T = TypeVar("T", bound="ContactListConsent")



@_attrs_define
class ContactListConsent:
    """ 
        Attributes:
            double_opt_in (bool):
            events (list[ContactConsentEvent]):
            list_id (str):
            name (str):
            status (ContactListConsentStatus):
     """

    double_opt_in: bool
    events: list[ContactConsentEvent]
    list_id: str
    name: str
    status: ContactListConsentStatus





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_consent_event import ContactConsentEvent # noqa: PLC0415
        double_opt_in = self.double_opt_in

        events = []
        for events_item_data in self.events:
            events_item = events_item_data.to_dict()
            events.append(events_item)



        list_id = self.list_id

        name = self.name

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "doubleOptIn": double_opt_in,
            "events": events,
            "listId": list_id,
            "name": name,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_consent_event import ContactConsentEvent # noqa: PLC0415
        d = dict(src_dict)
        double_opt_in = d.pop("doubleOptIn")

        events = []
        _events = d.pop("events")
        for events_item_data in (_events):
            events_item = ContactConsentEvent.from_dict(events_item_data)



            events.append(events_item)


        list_id = d.pop("listId")

        name = d.pop("name")

        status = ContactListConsentStatus(d.pop("status"))




        contact_list_consent = cls(
            double_opt_in=double_opt_in,
            events=events,
            list_id=list_id,
            name=name,
            status=status,
        )

        return contact_list_consent

