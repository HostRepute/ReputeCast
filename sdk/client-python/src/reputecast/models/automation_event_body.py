from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.automation_event_body_properties import AutomationEventBodyProperties





T = TypeVar("T", bound="AutomationEventBody")



@_attrs_define
class AutomationEventBody:
    """ 
        Attributes:
            event (str):  Example: workspace.created.
            contact_id (str | Unset):
            email (str | Unset):
            properties (AutomationEventBodyProperties | Unset):
     """

    event: str
    contact_id: str | Unset = UNSET
    email: str | Unset = UNSET
    properties: AutomationEventBodyProperties | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_event_body_properties import AutomationEventBodyProperties # noqa: PLC0415
        event = self.event

        contact_id = self.contact_id

        email = self.email

        properties: dict[str, Any] | Unset = UNSET
        if not isinstance(self.properties, Unset):
            properties = self.properties.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "event": event,
        })
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if email is not UNSET:
            field_dict["email"] = email
        if properties is not UNSET:
            field_dict["properties"] = properties

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_event_body_properties import AutomationEventBodyProperties # noqa: PLC0415
        d = dict(src_dict)
        event = d.pop("event")

        contact_id = d.pop("contactId", UNSET)

        email = d.pop("email", UNSET)

        _properties = d.pop("properties", UNSET)
        properties: AutomationEventBodyProperties | Unset
        if isinstance(_properties,  Unset):
            properties = UNSET
        else:
            properties = AutomationEventBodyProperties.from_dict(_properties)




        automation_event_body = cls(
            event=event,
            contact_id=contact_id,
            email=email,
            properties=properties,
        )

        return automation_event_body

