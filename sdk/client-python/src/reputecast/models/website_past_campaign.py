from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_past_campaign_note import WebsitePastCampaignNote
from ..models.website_past_campaign_visibility import WebsitePastCampaignVisibility
from typing import cast
import datetime






T = TypeVar("T", bound="WebsitePastCampaign")



@_attrs_define
class WebsitePastCampaign:
    """ 
        Attributes:
            id (str):
            lists (list[str]):
            name (str):
            note (WebsitePastCampaignNote):
            recipients (int):
            sent_at (datetime.datetime):
            subject (str):
            visibility (WebsitePastCampaignVisibility):
     """

    id: str
    lists: list[str]
    name: str
    note: WebsitePastCampaignNote
    recipients: int
    sent_at: datetime.datetime
    subject: str
    visibility: WebsitePastCampaignVisibility





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        lists = self.lists



        name = self.name

        note = self.note.value

        recipients = self.recipients

        sent_at = self.sent_at.isoformat()

        subject = self.subject

        visibility = self.visibility.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "lists": lists,
            "name": name,
            "note": note,
            "recipients": recipients,
            "sentAt": sent_at,
            "subject": subject,
            "visibility": visibility,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        lists = cast(list[str], d.pop("lists"))


        name = d.pop("name")

        note = WebsitePastCampaignNote(d.pop("note"))




        recipients = d.pop("recipients")

        sent_at = datetime.datetime.fromisoformat(d.pop("sentAt"))




        subject = d.pop("subject")

        visibility = WebsitePastCampaignVisibility(d.pop("visibility"))




        website_past_campaign = cls(
            id=id,
            lists=lists,
            name=name,
            note=note,
            recipients=recipients,
            sent_at=sent_at,
            subject=subject,
            visibility=visibility,
        )

        return website_past_campaign

