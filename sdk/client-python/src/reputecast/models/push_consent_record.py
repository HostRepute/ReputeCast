from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PushConsentRecord")



@_attrs_define
class PushConsentRecord:
    """ 
        Attributes:
            at (datetime.datetime):
            browser (str):
            page (str):
            site (str):
            text (str):
            time_zone (str):
            version (int):
            contact (str | Unset):
            contact_id (str | Unset):
            ip (str | Unset):
            linked_by (str | Unset):
            published_at (datetime.datetime | Unset):
            published_by (str | Unset):
     """

    at: datetime.datetime
    browser: str
    page: str
    site: str
    text: str
    time_zone: str
    version: int
    contact: str | Unset = UNSET
    contact_id: str | Unset = UNSET
    ip: str | Unset = UNSET
    linked_by: str | Unset = UNSET
    published_at: datetime.datetime | Unset = UNSET
    published_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        browser = self.browser

        page = self.page

        site = self.site

        text = self.text

        time_zone = self.time_zone

        version = self.version

        contact = self.contact

        contact_id = self.contact_id

        ip = self.ip

        linked_by = self.linked_by

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()

        published_by = self.published_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "browser": browser,
            "page": page,
            "site": site,
            "text": text,
            "timeZone": time_zone,
            "version": version,
        })
        if contact is not UNSET:
            field_dict["contact"] = contact
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if ip is not UNSET:
            field_dict["ip"] = ip
        if linked_by is not UNSET:
            field_dict["linkedBy"] = linked_by
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at
        if published_by is not UNSET:
            field_dict["publishedBy"] = published_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        browser = d.pop("browser")

        page = d.pop("page")

        site = d.pop("site")

        text = d.pop("text")

        time_zone = d.pop("timeZone")

        version = d.pop("version")

        contact = d.pop("contact", UNSET)

        contact_id = d.pop("contactId", UNSET)

        ip = d.pop("ip", UNSET)

        linked_by = d.pop("linkedBy", UNSET)

        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        published_by = d.pop("publishedBy", UNSET)

        push_consent_record = cls(
            at=at,
            browser=browser,
            page=page,
            site=site,
            text=text,
            time_zone=time_zone,
            version=version,
            contact=contact,
            contact_id=contact_id,
            ip=ip,
            linked_by=linked_by,
            published_at=published_at,
            published_by=published_by,
        )

        return push_consent_record

