from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.push_button import PushButton





T = TypeVar("T", bound="PushCampaignUpdate")



@_attrs_define
class PushCampaignUpdate:
    """ 
        Attributes:
            buttons (list[PushButton]):
            link (str):
            lists (list[str]):
            message (str):
            name (str):
            replace_tag (bool):
            segments (list[str]):
            site_ids (list[str]):
            tag (str):
            title (str):
            ttl_minutes (int):
            version (int):
            image_media_id (str | Unset):
     """

    buttons: list[PushButton]
    link: str
    lists: list[str]
    message: str
    name: str
    replace_tag: bool
    segments: list[str]
    site_ids: list[str]
    tag: str
    title: str
    ttl_minutes: int
    version: int
    image_media_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_button import PushButton # noqa: PLC0415
        buttons = []
        for buttons_item_data in self.buttons:
            buttons_item = buttons_item_data.to_dict()
            buttons.append(buttons_item)



        link = self.link

        lists = self.lists



        message = self.message

        name = self.name

        replace_tag = self.replace_tag

        segments = self.segments



        site_ids = self.site_ids



        tag = self.tag

        title = self.title

        ttl_minutes = self.ttl_minutes

        version = self.version

        image_media_id = self.image_media_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "buttons": buttons,
            "link": link,
            "lists": lists,
            "message": message,
            "name": name,
            "replaceTag": replace_tag,
            "segments": segments,
            "siteIds": site_ids,
            "tag": tag,
            "title": title,
            "ttlMinutes": ttl_minutes,
            "version": version,
        })
        if image_media_id is not UNSET:
            field_dict["imageMediaId"] = image_media_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_button import PushButton # noqa: PLC0415
        d = dict(src_dict)
        buttons = []
        _buttons = d.pop("buttons")
        for buttons_item_data in (_buttons):
            buttons_item = PushButton.from_dict(buttons_item_data)



            buttons.append(buttons_item)


        link = d.pop("link")

        lists = cast(list[str], d.pop("lists"))


        message = d.pop("message")

        name = d.pop("name")

        replace_tag = d.pop("replaceTag")

        segments = cast(list[str], d.pop("segments"))


        site_ids = cast(list[str], d.pop("siteIds"))


        tag = d.pop("tag")

        title = d.pop("title")

        ttl_minutes = d.pop("ttlMinutes")

        version = d.pop("version")

        image_media_id = d.pop("imageMediaId", UNSET)

        push_campaign_update = cls(
            buttons=buttons,
            link=link,
            lists=lists,
            message=message,
            name=name,
            replace_tag=replace_tag,
            segments=segments,
            site_ids=site_ids,
            tag=tag,
            title=title,
            ttl_minutes=ttl_minutes,
            version=version,
            image_media_id=image_media_id,
        )

        return push_campaign_update

