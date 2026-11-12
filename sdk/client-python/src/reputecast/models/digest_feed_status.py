from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_feed_status_kind import DigestFeedStatusKind
from ..models.digest_feed_status_status import DigestFeedStatusStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.digest_feed_error import DigestFeedError





T = TypeVar("T", bound="DigestFeedStatus")



@_attrs_define
class DigestFeedStatus:
    """ 
        Attributes:
            bytes_ (int):
            items (int):
            left_out (int):
            status (DigestFeedStatusStatus):
            title (str):
            url (str):
            waiting (int):
            error (DigestFeedError | Unset):
            kind (DigestFeedStatusKind | Unset):
            last_ok_at (datetime.datetime | Unset):
            last_read_at (datetime.datetime | Unset):
            last_read_via (str | Unset):
            moved_to (str | Unset):
     """

    bytes_: int
    items: int
    left_out: int
    status: DigestFeedStatusStatus
    title: str
    url: str
    waiting: int
    error: DigestFeedError | Unset = UNSET
    kind: DigestFeedStatusKind | Unset = UNSET
    last_ok_at: datetime.datetime | Unset = UNSET
    last_read_at: datetime.datetime | Unset = UNSET
    last_read_via: str | Unset = UNSET
    moved_to: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_feed_error import DigestFeedError # noqa: PLC0415
        bytes_ = self.bytes_

        items = self.items

        left_out = self.left_out

        status = self.status.value

        title = self.title

        url = self.url

        waiting = self.waiting

        error: dict[str, Any] | Unset = UNSET
        if not isinstance(self.error, Unset):
            error = self.error.to_dict()

        kind: str | Unset = UNSET
        if not isinstance(self.kind, Unset):
            kind = self.kind.value


        last_ok_at: str | Unset = UNSET
        if not isinstance(self.last_ok_at, Unset):
            last_ok_at = self.last_ok_at.isoformat()

        last_read_at: str | Unset = UNSET
        if not isinstance(self.last_read_at, Unset):
            last_read_at = self.last_read_at.isoformat()

        last_read_via = self.last_read_via

        moved_to = self.moved_to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bytes": bytes_,
            "items": items,
            "leftOut": left_out,
            "status": status,
            "title": title,
            "url": url,
            "waiting": waiting,
        })
        if error is not UNSET:
            field_dict["error"] = error
        if kind is not UNSET:
            field_dict["kind"] = kind
        if last_ok_at is not UNSET:
            field_dict["lastOkAt"] = last_ok_at
        if last_read_at is not UNSET:
            field_dict["lastReadAt"] = last_read_at
        if last_read_via is not UNSET:
            field_dict["lastReadVia"] = last_read_via
        if moved_to is not UNSET:
            field_dict["movedTo"] = moved_to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_feed_error import DigestFeedError # noqa: PLC0415
        d = dict(src_dict)
        bytes_ = d.pop("bytes")

        items = d.pop("items")

        left_out = d.pop("leftOut")

        status = DigestFeedStatusStatus(d.pop("status"))




        title = d.pop("title")

        url = d.pop("url")

        waiting = d.pop("waiting")

        _error = d.pop("error", UNSET)
        error: DigestFeedError | Unset
        if isinstance(_error,  Unset):
            error = UNSET
        else:
            error = DigestFeedError.from_dict(_error)




        _kind = d.pop("kind", UNSET)
        kind: DigestFeedStatusKind | Unset
        if isinstance(_kind,  Unset):
            kind = UNSET
        else:
            kind = DigestFeedStatusKind(_kind)




        _last_ok_at = d.pop("lastOkAt", UNSET)
        last_ok_at: datetime.datetime | Unset
        if isinstance(_last_ok_at,  Unset):
            last_ok_at = UNSET
        else:
            last_ok_at = datetime.datetime.fromisoformat(_last_ok_at)




        _last_read_at = d.pop("lastReadAt", UNSET)
        last_read_at: datetime.datetime | Unset
        if isinstance(_last_read_at,  Unset):
            last_read_at = UNSET
        else:
            last_read_at = datetime.datetime.fromisoformat(_last_read_at)




        last_read_via = d.pop("lastReadVia", UNSET)

        moved_to = d.pop("movedTo", UNSET)

        digest_feed_status = cls(
            bytes_=bytes_,
            items=items,
            left_out=left_out,
            status=status,
            title=title,
            url=url,
            waiting=waiting,
            error=error,
            kind=kind,
            last_ok_at=last_ok_at,
            last_read_at=last_read_at,
            last_read_via=last_read_via,
            moved_to=moved_to,
        )

        return digest_feed_status

