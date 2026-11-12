from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.page_reminder_flow import PageReminderFlow





T = TypeVar("T", bound="PageReminders")



@_attrs_define
class PageReminders:
    """ 
        Attributes:
            available (bool):
            field (str):
            reminders (str):
            title (str):
            automation (PageReminderFlow | Unset):
     """

    available: bool
    field: str
    reminders: str
    title: str
    automation: PageReminderFlow | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.page_reminder_flow import PageReminderFlow # noqa: PLC0415
        available = self.available

        field = self.field

        reminders = self.reminders

        title = self.title

        automation: dict[str, Any] | Unset = UNSET
        if not isinstance(self.automation, Unset):
            automation = self.automation.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
            "field": field,
            "reminders": reminders,
            "title": title,
        })
        if automation is not UNSET:
            field_dict["automation"] = automation

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.page_reminder_flow import PageReminderFlow # noqa: PLC0415
        d = dict(src_dict)
        available = d.pop("available")

        field = d.pop("field")

        reminders = d.pop("reminders")

        title = d.pop("title")

        _automation = d.pop("automation", UNSET)
        automation: PageReminderFlow | Unset
        if isinstance(_automation,  Unset):
            automation = UNSET
        else:
            automation = PageReminderFlow.from_dict(_automation)




        page_reminders = cls(
            available=available,
            field=field,
            reminders=reminders,
            title=title,
            automation=automation,
        )

        return page_reminders

