from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.approval_push_button import ApprovalPushButton





T = TypeVar("T", bound="ApprovalPushSnapshot")



@_attrs_define
class ApprovalPushSnapshot:
    """ 
        Attributes:
            buttons (list[ApprovalPushButton]):
            link (str):
            message (str):
            recipients (int):
            schedule (str):
            sites (list[str]):
            title (str):
            ttl_minutes (int):
            host (str | Unset):
            icon (str | Unset):
            image (str | Unset):
            tag (str | Unset):
     """

    buttons: list[ApprovalPushButton]
    link: str
    message: str
    recipients: int
    schedule: str
    sites: list[str]
    title: str
    ttl_minutes: int
    host: str | Unset = UNSET
    icon: str | Unset = UNSET
    image: str | Unset = UNSET
    tag: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_push_button import ApprovalPushButton # noqa: PLC0415
        buttons = []
        for buttons_item_data in self.buttons:
            buttons_item = buttons_item_data.to_dict()
            buttons.append(buttons_item)



        link = self.link

        message = self.message

        recipients = self.recipients

        schedule = self.schedule

        sites = self.sites



        title = self.title

        ttl_minutes = self.ttl_minutes

        host = self.host

        icon = self.icon

        image = self.image

        tag = self.tag


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "buttons": buttons,
            "link": link,
            "message": message,
            "recipients": recipients,
            "schedule": schedule,
            "sites": sites,
            "title": title,
            "ttlMinutes": ttl_minutes,
        })
        if host is not UNSET:
            field_dict["host"] = host
        if icon is not UNSET:
            field_dict["icon"] = icon
        if image is not UNSET:
            field_dict["image"] = image
        if tag is not UNSET:
            field_dict["tag"] = tag

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_push_button import ApprovalPushButton # noqa: PLC0415
        d = dict(src_dict)
        buttons = []
        _buttons = d.pop("buttons")
        for buttons_item_data in (_buttons):
            buttons_item = ApprovalPushButton.from_dict(buttons_item_data)



            buttons.append(buttons_item)


        link = d.pop("link")

        message = d.pop("message")

        recipients = d.pop("recipients")

        schedule = d.pop("schedule")

        sites = cast(list[str], d.pop("sites"))


        title = d.pop("title")

        ttl_minutes = d.pop("ttlMinutes")

        host = d.pop("host", UNSET)

        icon = d.pop("icon", UNSET)

        image = d.pop("image", UNSET)

        tag = d.pop("tag", UNSET)

        approval_push_snapshot = cls(
            buttons=buttons,
            link=link,
            message=message,
            recipients=recipients,
            schedule=schedule,
            sites=sites,
            title=title,
            ttl_minutes=ttl_minutes,
            host=host,
            icon=icon,
            image=image,
            tag=tag,
        )

        return approval_push_snapshot

