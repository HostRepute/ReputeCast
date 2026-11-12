from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.digest_feed_error import DigestFeedError
  from ..models.digest_feed_hop import DigestFeedHop
  from ..models.digest_feed_item import DigestFeedItem





T = TypeVar("T", bound="DigestFound")



@_attrs_define
class DigestFound:
    """ 
        Attributes:
            bytes_ (int):
            hops (list[DigestFeedHop]):
            items (int):
            ok (bool):
            preview (list[DigestFeedItem]):
            error (DigestFeedError | Unset):
            kind (str | Unset):
            moved (str | Unset):
            page (str | Unset):
            title (str | Unset):
            url (str | Unset):
     """

    bytes_: int
    hops: list[DigestFeedHop]
    items: int
    ok: bool
    preview: list[DigestFeedItem]
    error: DigestFeedError | Unset = UNSET
    kind: str | Unset = UNSET
    moved: str | Unset = UNSET
    page: str | Unset = UNSET
    title: str | Unset = UNSET
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_feed_error import DigestFeedError # noqa: PLC0415
        from ..models.digest_feed_hop import DigestFeedHop # noqa: PLC0415
        from ..models.digest_feed_item import DigestFeedItem # noqa: PLC0415
        bytes_ = self.bytes_

        hops = []
        for hops_item_data in self.hops:
            hops_item = hops_item_data.to_dict()
            hops.append(hops_item)



        items = self.items

        ok = self.ok

        preview = []
        for preview_item_data in self.preview:
            preview_item = preview_item_data.to_dict()
            preview.append(preview_item)



        error: dict[str, Any] | Unset = UNSET
        if not isinstance(self.error, Unset):
            error = self.error.to_dict()

        kind = self.kind

        moved = self.moved

        page = self.page

        title = self.title

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bytes": bytes_,
            "hops": hops,
            "items": items,
            "ok": ok,
            "preview": preview,
        })
        if error is not UNSET:
            field_dict["error"] = error
        if kind is not UNSET:
            field_dict["kind"] = kind
        if moved is not UNSET:
            field_dict["moved"] = moved
        if page is not UNSET:
            field_dict["page"] = page
        if title is not UNSET:
            field_dict["title"] = title
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_feed_error import DigestFeedError # noqa: PLC0415
        from ..models.digest_feed_hop import DigestFeedHop # noqa: PLC0415
        from ..models.digest_feed_item import DigestFeedItem # noqa: PLC0415
        d = dict(src_dict)
        bytes_ = d.pop("bytes")

        hops = []
        _hops = d.pop("hops")
        for hops_item_data in (_hops):
            hops_item = DigestFeedHop.from_dict(hops_item_data)



            hops.append(hops_item)


        items = d.pop("items")

        ok = d.pop("ok")

        preview = []
        _preview = d.pop("preview")
        for preview_item_data in (_preview):
            preview_item = DigestFeedItem.from_dict(preview_item_data)



            preview.append(preview_item)


        _error = d.pop("error", UNSET)
        error: DigestFeedError | Unset
        if isinstance(_error,  Unset):
            error = UNSET
        else:
            error = DigestFeedError.from_dict(_error)




        kind = d.pop("kind", UNSET)

        moved = d.pop("moved", UNSET)

        page = d.pop("page", UNSET)

        title = d.pop("title", UNSET)

        url = d.pop("url", UNSET)

        digest_found = cls(
            bytes_=bytes_,
            hops=hops,
            items=items,
            ok=ok,
            preview=preview,
            error=error,
            kind=kind,
            moved=moved,
            page=page,
            title=title,
            url=url,
        )

        return digest_found

