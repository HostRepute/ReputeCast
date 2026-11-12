from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ContentCheckImage")



@_attrs_define
class ContentCheckImage:
    """ 
        Attributes:
            path (list[int]):
            url (str):
            broken (bool | Unset):
            bytes_ (int | Unset):
            host (str | Unset):
            insecure (bool | Unset):
            probed (bool | Unset):
            status (int | Unset):
     """

    path: list[int]
    url: str
    broken: bool | Unset = UNSET
    bytes_: int | Unset = UNSET
    host: str | Unset = UNSET
    insecure: bool | Unset = UNSET
    probed: bool | Unset = UNSET
    status: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        path = self.path



        url = self.url

        broken = self.broken

        bytes_ = self.bytes_

        host = self.host

        insecure = self.insecure

        probed = self.probed

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "path": path,
            "url": url,
        })
        if broken is not UNSET:
            field_dict["broken"] = broken
        if bytes_ is not UNSET:
            field_dict["bytes"] = bytes_
        if host is not UNSET:
            field_dict["host"] = host
        if insecure is not UNSET:
            field_dict["insecure"] = insecure
        if probed is not UNSET:
            field_dict["probed"] = probed
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        path = cast(list[int], d.pop("path"))


        url = d.pop("url")

        broken = d.pop("broken", UNSET)

        bytes_ = d.pop("bytes", UNSET)

        host = d.pop("host", UNSET)

        insecure = d.pop("insecure", UNSET)

        probed = d.pop("probed", UNSET)

        status = d.pop("status", UNSET)

        content_check_image = cls(
            path=path,
            url=url,
            broken=broken,
            bytes_=bytes_,
            host=host,
            insecure=insecure,
            probed=probed,
            status=status,
        )

        return content_check_image

