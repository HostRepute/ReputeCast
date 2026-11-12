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





T = TypeVar("T", bound="DigestReadNow")



@_attrs_define
class DigestReadNow:
    """ 
        Attributes:
            items (int):
            new (int):
            not_modified (bool):
            ok (bool):
            error (DigestFeedError | Unset):
     """

    items: int
    new: int
    not_modified: bool
    ok: bool
    error: DigestFeedError | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_feed_error import DigestFeedError # noqa: PLC0415
        items = self.items

        new = self.new

        not_modified = self.not_modified

        ok = self.ok

        error: dict[str, Any] | Unset = UNSET
        if not isinstance(self.error, Unset):
            error = self.error.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "new": new,
            "notModified": not_modified,
            "ok": ok,
        })
        if error is not UNSET:
            field_dict["error"] = error

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_feed_error import DigestFeedError # noqa: PLC0415
        d = dict(src_dict)
        items = d.pop("items")

        new = d.pop("new")

        not_modified = d.pop("notModified")

        ok = d.pop("ok")

        _error = d.pop("error", UNSET)
        error: DigestFeedError | Unset
        if isinstance(_error,  Unset):
            error = UNSET
        else:
            error = DigestFeedError.from_dict(_error)




        digest_read_now = cls(
            items=items,
            new=new,
            not_modified=not_modified,
            ok=ok,
            error=error,
        )

        return digest_read_now

