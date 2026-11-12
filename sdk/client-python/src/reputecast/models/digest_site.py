from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_site_status import DigestSiteStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DigestSite")



@_attrs_define
class DigestSite:
    """ 
        Attributes:
            connected_at (datetime.datetime):
            connected_by (str):
            draft_categories (list[str]):
            feed_url (str):
            host (str):
            id (str):
            items (int):
            latest_plugin (str):
            name (str):
            plugin_version (str):
            status (DigestSiteStatus):
            draft_list (str | Unset):
            last_report_at (datetime.datetime | Unset):
     """

    connected_at: datetime.datetime
    connected_by: str
    draft_categories: list[str]
    feed_url: str
    host: str
    id: str
    items: int
    latest_plugin: str
    name: str
    plugin_version: str
    status: DigestSiteStatus
    draft_list: str | Unset = UNSET
    last_report_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        connected_at = self.connected_at.isoformat()

        connected_by = self.connected_by

        draft_categories = self.draft_categories



        feed_url = self.feed_url

        host = self.host

        id = self.id

        items = self.items

        latest_plugin = self.latest_plugin

        name = self.name

        plugin_version = self.plugin_version

        status = self.status.value

        draft_list = self.draft_list

        last_report_at: str | Unset = UNSET
        if not isinstance(self.last_report_at, Unset):
            last_report_at = self.last_report_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "connectedAt": connected_at,
            "connectedBy": connected_by,
            "draftCategories": draft_categories,
            "feedUrl": feed_url,
            "host": host,
            "id": id,
            "items": items,
            "latestPlugin": latest_plugin,
            "name": name,
            "pluginVersion": plugin_version,
            "status": status,
        })
        if draft_list is not UNSET:
            field_dict["draftList"] = draft_list
        if last_report_at is not UNSET:
            field_dict["lastReportAt"] = last_report_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        connected_at = datetime.datetime.fromisoformat(d.pop("connectedAt"))




        connected_by = d.pop("connectedBy")

        draft_categories = cast(list[str], d.pop("draftCategories"))


        feed_url = d.pop("feedUrl")

        host = d.pop("host")

        id = d.pop("id")

        items = d.pop("items")

        latest_plugin = d.pop("latestPlugin")

        name = d.pop("name")

        plugin_version = d.pop("pluginVersion")

        status = DigestSiteStatus(d.pop("status"))




        draft_list = d.pop("draftList", UNSET)

        _last_report_at = d.pop("lastReportAt", UNSET)
        last_report_at: datetime.datetime | Unset
        if isinstance(_last_report_at,  Unset):
            last_report_at = UNSET
        else:
            last_report_at = datetime.datetime.fromisoformat(_last_report_at)




        digest_site = cls(
            connected_at=connected_at,
            connected_by=connected_by,
            draft_categories=draft_categories,
            feed_url=feed_url,
            host=host,
            id=id,
            items=items,
            latest_plugin=latest_plugin,
            name=name,
            plugin_version=plugin_version,
            status=status,
            draft_list=draft_list,
            last_report_at=last_report_at,
        )

        return digest_site

