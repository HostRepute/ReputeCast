from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.digest_feed_hop import DigestFeedHop





T = TypeVar("T", bound="DigestFeedError")



@_attrs_define
class DigestFeedError:
    """ 
        Attributes:
            detail (str):
            hops (list[DigestFeedHop]):
            host (str):
            kind (str):
            lines (list[str]):
            url (str):
            seconds (int | Unset):
            status (int | Unset):
     """

    detail: str
    hops: list[DigestFeedHop]
    host: str
    kind: str
    lines: list[str]
    url: str
    seconds: int | Unset = UNSET
    status: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_feed_hop import DigestFeedHop # noqa: PLC0415
        detail = self.detail

        hops = []
        for hops_item_data in self.hops:
            hops_item = hops_item_data.to_dict()
            hops.append(hops_item)



        host = self.host

        kind = self.kind

        lines = self.lines



        url = self.url

        seconds = self.seconds

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "detail": detail,
            "hops": hops,
            "host": host,
            "kind": kind,
            "lines": lines,
            "url": url,
        })
        if seconds is not UNSET:
            field_dict["seconds"] = seconds
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_feed_hop import DigestFeedHop # noqa: PLC0415
        d = dict(src_dict)
        detail = d.pop("detail")

        hops = []
        _hops = d.pop("hops")
        for hops_item_data in (_hops):
            hops_item = DigestFeedHop.from_dict(hops_item_data)



            hops.append(hops_item)


        host = d.pop("host")

        kind = d.pop("kind")

        lines = cast(list[str], d.pop("lines"))


        url = d.pop("url")

        seconds = d.pop("seconds", UNSET)

        status = d.pop("status", UNSET)

        digest_feed_error = cls(
            detail=detail,
            hops=hops,
            host=host,
            kind=kind,
            lines=lines,
            url=url,
            seconds=seconds,
            status=status,
        )

        return digest_feed_error

