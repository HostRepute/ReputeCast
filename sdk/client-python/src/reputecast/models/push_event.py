from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_event_kind import PushEventKind
from ..models.push_event_source_kind import PushEventSourceKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PushEvent")



@_attrs_define
class PushEvent:
    """ 
        Attributes:
            at (datetime.datetime):
            browser (str):
            kind (PushEventKind):
            message (str):
            raw (str):
            site (str):
            source (str):
            source_kind (PushEventSourceKind):
            source_id (str | Unset):
     """

    at: datetime.datetime
    browser: str
    kind: PushEventKind
    message: str
    raw: str
    site: str
    source: str
    source_kind: PushEventSourceKind
    source_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        browser = self.browser

        kind = self.kind.value

        message = self.message

        raw = self.raw

        site = self.site

        source = self.source

        source_kind = self.source_kind.value

        source_id = self.source_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "browser": browser,
            "kind": kind,
            "message": message,
            "raw": raw,
            "site": site,
            "source": source,
            "sourceKind": source_kind,
        })
        if source_id is not UNSET:
            field_dict["sourceId"] = source_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        browser = d.pop("browser")

        kind = PushEventKind(d.pop("kind"))




        message = d.pop("message")

        raw = d.pop("raw")

        site = d.pop("site")

        source = d.pop("source")

        source_kind = PushEventSourceKind(d.pop("sourceKind"))




        source_id = d.pop("sourceId", UNSET)

        push_event = cls(
            at=at,
            browser=browser,
            kind=kind,
            message=message,
            raw=raw,
            site=site,
            source=source,
            source_kind=source_kind,
            source_id=source_id,
        )

        return push_event

