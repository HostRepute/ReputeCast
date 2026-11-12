from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ArchiveIssueItem")



@_attrs_define
class ArchiveIssueItem:
    """ 
        Attributes:
            campaign_id (str):
            excerpt (str):
            excerpt_override (str):
            hidden (bool):
            label (str):
            list_id (str):
            list_name (str):
            sent_at (datetime.datetime):
            slug (str):
            subject (str):
            title (str):
            title_override (str):
            url (str):
     """

    campaign_id: str
    excerpt: str
    excerpt_override: str
    hidden: bool
    label: str
    list_id: str
    list_name: str
    sent_at: datetime.datetime
    slug: str
    subject: str
    title: str
    title_override: str
    url: str





    def to_dict(self) -> dict[str, Any]:
        campaign_id = self.campaign_id

        excerpt = self.excerpt

        excerpt_override = self.excerpt_override

        hidden = self.hidden

        label = self.label

        list_id = self.list_id

        list_name = self.list_name

        sent_at = self.sent_at.isoformat()

        slug = self.slug

        subject = self.subject

        title = self.title

        title_override = self.title_override

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaignId": campaign_id,
            "excerpt": excerpt,
            "excerptOverride": excerpt_override,
            "hidden": hidden,
            "label": label,
            "listId": list_id,
            "listName": list_name,
            "sentAt": sent_at,
            "slug": slug,
            "subject": subject,
            "title": title,
            "titleOverride": title_override,
            "url": url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        campaign_id = d.pop("campaignId")

        excerpt = d.pop("excerpt")

        excerpt_override = d.pop("excerptOverride")

        hidden = d.pop("hidden")

        label = d.pop("label")

        list_id = d.pop("listId")

        list_name = d.pop("listName")

        sent_at = datetime.datetime.fromisoformat(d.pop("sentAt"))




        slug = d.pop("slug")

        subject = d.pop("subject")

        title = d.pop("title")

        title_override = d.pop("titleOverride")

        url = d.pop("url")

        archive_issue_item = cls(
            campaign_id=campaign_id,
            excerpt=excerpt,
            excerpt_override=excerpt_override,
            hidden=hidden,
            label=label,
            list_id=list_id,
            list_name=list_name,
            sent_at=sent_at,
            slug=slug,
            subject=subject,
            title=title,
            title_override=title_override,
            url=url,
        )

        return archive_issue_item

