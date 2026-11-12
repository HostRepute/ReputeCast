from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_thread_item_direction import WhatsAppThreadItemDirection
from ..models.whats_app_thread_item_kind import WhatsAppThreadItemKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppThreadItem")



@_attrs_define
class WhatsAppThreadItem:
    """ 
        Attributes:
            at (datetime.datetime):
            direction (WhatsAppThreadItemDirection):
            id (str):
            kind (WhatsAppThreadItemKind):
            read (bool):
            text (str):
            campaign (str | Unset):
            status (str | Unset):
            template (str | Unset):
     """

    at: datetime.datetime
    direction: WhatsAppThreadItemDirection
    id: str
    kind: WhatsAppThreadItemKind
    read: bool
    text: str
    campaign: str | Unset = UNSET
    status: str | Unset = UNSET
    template: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        direction = self.direction.value

        id = self.id

        kind = self.kind.value

        read = self.read

        text = self.text

        campaign = self.campaign

        status = self.status

        template = self.template


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "direction": direction,
            "id": id,
            "kind": kind,
            "read": read,
            "text": text,
        })
        if campaign is not UNSET:
            field_dict["campaign"] = campaign
        if status is not UNSET:
            field_dict["status"] = status
        if template is not UNSET:
            field_dict["template"] = template

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        direction = WhatsAppThreadItemDirection(d.pop("direction"))




        id = d.pop("id")

        kind = WhatsAppThreadItemKind(d.pop("kind"))




        read = d.pop("read")

        text = d.pop("text")

        campaign = d.pop("campaign", UNSET)

        status = d.pop("status", UNSET)

        template = d.pop("template", UNSET)

        whats_app_thread_item = cls(
            at=at,
            direction=direction,
            id=id,
            kind=kind,
            read=read,
            text=text,
            campaign=campaign,
            status=status,
            template=template,
        )

        return whats_app_thread_item

