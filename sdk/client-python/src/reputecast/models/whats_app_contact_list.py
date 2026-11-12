from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_contact_list_status import WhatsAppContactListStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppContactList")



@_attrs_define
class WhatsAppContactList:
    """ 
        Attributes:
            list_id (str):
            name (str):
            source (str):
            status (WhatsAppContactListStatus):
            at (datetime.datetime | Unset):
            keyword (str | Unset):
            page (str | Unset):
            version (str | Unset):
     """

    list_id: str
    name: str
    source: str
    status: WhatsAppContactListStatus
    at: datetime.datetime | Unset = UNSET
    keyword: str | Unset = UNSET
    page: str | Unset = UNSET
    version: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        list_id = self.list_id

        name = self.name

        source = self.source

        status = self.status.value

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        keyword = self.keyword

        page = self.page

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "listId": list_id,
            "name": name,
            "source": source,
            "status": status,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if keyword is not UNSET:
            field_dict["keyword"] = keyword
        if page is not UNSET:
            field_dict["page"] = page
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        list_id = d.pop("listId")

        name = d.pop("name")

        source = d.pop("source")

        status = WhatsAppContactListStatus(d.pop("status"))




        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        keyword = d.pop("keyword", UNSET)

        page = d.pop("page", UNSET)

        version = d.pop("version", UNSET)

        whats_app_contact_list = cls(
            list_id=list_id,
            name=name,
            source=source,
            status=status,
            at=at,
            keyword=keyword,
            page=page,
            version=version,
        )

        return whats_app_contact_list

