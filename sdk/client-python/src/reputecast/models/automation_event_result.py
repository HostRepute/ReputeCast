from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AutomationEventResult")



@_attrs_define
class AutomationEventResult:
    """ 
        Attributes:
            accepted (bool):
            contact_id (str):
            flows (int): Live automations triggered by this event
     """

    accepted: bool
    contact_id: str
    flows: int





    def to_dict(self) -> dict[str, Any]:
        accepted = self.accepted

        contact_id = self.contact_id

        flows = self.flows


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accepted": accepted,
            "contactId": contact_id,
            "flows": flows,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        accepted = d.pop("accepted")

        contact_id = d.pop("contactId")

        flows = d.pop("flows")

        automation_event_result = cls(
            accepted=accepted,
            contact_id=contact_id,
            flows=flows,
        )

        return automation_event_result

