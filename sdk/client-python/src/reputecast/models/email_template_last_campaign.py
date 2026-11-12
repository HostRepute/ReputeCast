from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.mjml_thumb_item import MJMLThumbItem





T = TypeVar("T", bound="EmailTemplateLastCampaign")



@_attrs_define
class EmailTemplateLastCampaign:
    """ 
        Attributes:
            id (str):
            name (str):
            sent_at (datetime.datetime):
            thumb (list[MJMLThumbItem]):
     """

    id: str
    name: str
    sent_at: datetime.datetime
    thumb: list[MJMLThumbItem]





    def to_dict(self) -> dict[str, Any]:
        from ..models.mjml_thumb_item import MJMLThumbItem # noqa: PLC0415
        id = self.id

        name = self.name

        sent_at = self.sent_at.isoformat()

        thumb = []
        for thumb_item_data in self.thumb:
            thumb_item = thumb_item_data.to_dict()
            thumb.append(thumb_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "sentAt": sent_at,
            "thumb": thumb,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.mjml_thumb_item import MJMLThumbItem # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        sent_at = datetime.datetime.fromisoformat(d.pop("sentAt"))




        thumb = []
        _thumb = d.pop("thumb")
        for thumb_item_data in (_thumb):
            thumb_item = MJMLThumbItem.from_dict(thumb_item_data)



            thumb.append(thumb_item)


        email_template_last_campaign = cls(
            id=id,
            name=name,
            sent_at=sent_at,
            thumb=thumb,
        )

        return email_template_last_campaign

