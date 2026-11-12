from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_check_site_status import PushCheckSiteStatus
from ..types import UNSET, Unset






T = TypeVar("T", bound="PushCheckSite")



@_attrs_define
class PushCheckSite:
    """ 
        Attributes:
            host (str):
            status (PushCheckSiteStatus):
            subscribers (int):
            url (str):
            http_status (int | Unset):
            version (int | Unset):
     """

    host: str
    status: PushCheckSiteStatus
    subscribers: int
    url: str
    http_status: int | Unset = UNSET
    version: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        host = self.host

        status = self.status.value

        subscribers = self.subscribers

        url = self.url

        http_status = self.http_status

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "host": host,
            "status": status,
            "subscribers": subscribers,
            "url": url,
        })
        if http_status is not UNSET:
            field_dict["httpStatus"] = http_status
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        host = d.pop("host")

        status = PushCheckSiteStatus(d.pop("status"))




        subscribers = d.pop("subscribers")

        url = d.pop("url")

        http_status = d.pop("httpStatus", UNSET)

        version = d.pop("version", UNSET)

        push_check_site = cls(
            host=host,
            status=status,
            subscribers=subscribers,
            url=url,
            http_status=http_status,
            version=version,
        )

        return push_check_site

