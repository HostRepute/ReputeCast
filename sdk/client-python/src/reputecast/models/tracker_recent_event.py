from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.tracker_recent_event_kind import TrackerRecentEventKind
from typing import cast
import datetime






T = TypeVar("T", bound="TrackerRecentEvent")



@_attrs_define
class TrackerRecentEvent:
    """ 
        Attributes:
            at (datetime.datetime):
            kind (TrackerRecentEventKind):
            known (bool): A known contact's, else a visitor's
            name (str):
            path (str):
            product_title (str):
     """

    at: datetime.datetime
    kind: TrackerRecentEventKind
    known: bool
    name: str
    path: str
    product_title: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        kind = self.kind.value

        known = self.known

        name = self.name

        path = self.path

        product_title = self.product_title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "kind": kind,
            "known": known,
            "name": name,
            "path": path,
            "productTitle": product_title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        kind = TrackerRecentEventKind(d.pop("kind"))




        known = d.pop("known")

        name = d.pop("name")

        path = d.pop("path")

        product_title = d.pop("productTitle")

        tracker_recent_event = cls(
            at=at,
            kind=kind,
            known=known,
            name=name,
            path=path,
            product_title=product_title,
        )

        return tracker_recent_event

