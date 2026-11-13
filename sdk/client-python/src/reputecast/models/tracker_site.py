from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
from uuid import UUID
import datetime

if TYPE_CHECKING:
  from ..models.tracker_refusal import TrackerRefusal
  from ..models.tracker_snippet import TrackerSnippet
  from ..models.tracker_store_option import TrackerStoreOption





T = TypeVar("T", bound="TrackerSite")



@_attrs_define
class TrackerSite:
    """ 
        Attributes:
            created_at (datetime.datetime):
            created_by (str):
            events_today (int): Events since midnight in the workspace's time zone
            has_identify_secret (bool):
            id (UUID):
            name (str):
            origin (str):  Example: https://shop.acme.io.
            refused (list[TrackerRefusal]): Origins refused in the last 24 hours, most refused first
            snippet (TrackerSnippet):
            wait_consent (bool):
            last_event_at (datetime.datetime | Unset):
            store (TrackerStoreOption | Unset):
     """

    created_at: datetime.datetime
    created_by: str
    events_today: int
    has_identify_secret: bool
    id: UUID
    name: str
    origin: str
    refused: list[TrackerRefusal]
    snippet: TrackerSnippet
    wait_consent: bool
    last_event_at: datetime.datetime | Unset = UNSET
    store: TrackerStoreOption | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.tracker_refusal import TrackerRefusal # noqa: PLC0415
        from ..models.tracker_snippet import TrackerSnippet # noqa: PLC0415
        from ..models.tracker_store_option import TrackerStoreOption # noqa: PLC0415
        created_at = self.created_at.isoformat()

        created_by = self.created_by

        events_today = self.events_today

        has_identify_secret = self.has_identify_secret

        id = str(self.id)

        name = self.name

        origin = self.origin

        refused = []
        for refused_item_data in self.refused:
            refused_item = refused_item_data.to_dict()
            refused.append(refused_item)



        snippet = self.snippet.to_dict()

        wait_consent = self.wait_consent

        last_event_at: str | Unset = UNSET
        if not isinstance(self.last_event_at, Unset):
            last_event_at = self.last_event_at.isoformat()

        store: dict[str, Any] | Unset = UNSET
        if not isinstance(self.store, Unset):
            store = self.store.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "createdBy": created_by,
            "eventsToday": events_today,
            "hasIdentifySecret": has_identify_secret,
            "id": id,
            "name": name,
            "origin": origin,
            "refused": refused,
            "snippet": snippet,
            "waitConsent": wait_consent,
        })
        if last_event_at is not UNSET:
            field_dict["lastEventAt"] = last_event_at
        if store is not UNSET:
            field_dict["store"] = store

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.tracker_refusal import TrackerRefusal # noqa: PLC0415
        from ..models.tracker_snippet import TrackerSnippet # noqa: PLC0415
        from ..models.tracker_store_option import TrackerStoreOption # noqa: PLC0415
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        events_today = d.pop("eventsToday")

        has_identify_secret = d.pop("hasIdentifySecret")

        id = UUID(d.pop("id"))




        name = d.pop("name")

        origin = d.pop("origin")

        refused = []
        _refused = d.pop("refused")
        for refused_item_data in (_refused):
            refused_item = TrackerRefusal.from_dict(refused_item_data)



            refused.append(refused_item)


        snippet = TrackerSnippet.from_dict(d.pop("snippet"))




        wait_consent = d.pop("waitConsent")

        _last_event_at = d.pop("lastEventAt", UNSET)
        last_event_at: datetime.datetime | Unset
        if isinstance(_last_event_at,  Unset):
            last_event_at = UNSET
        else:
            last_event_at = datetime.datetime.fromisoformat(_last_event_at)




        _store = d.pop("store", UNSET)
        store: TrackerStoreOption | Unset
        if isinstance(_store,  Unset):
            store = UNSET
        else:
            store = TrackerStoreOption.from_dict(_store)




        tracker_site = cls(
            created_at=created_at,
            created_by=created_by,
            events_today=events_today,
            has_identify_secret=has_identify_secret,
            id=id,
            name=name,
            origin=origin,
            refused=refused,
            snippet=snippet,
            wait_consent=wait_consent,
            last_event_at=last_event_at,
            store=store,
        )

        return tracker_site

