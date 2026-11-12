from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_site_kind import PushSiteKind
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.push_site_prompt import PushSitePrompt
  from ..models.push_worker import PushWorker





T = TypeVar("T", bound="PushSite")



@_attrs_define
class PushSite:
    """ 
        Attributes:
            added_at (datetime.datetime):
            added_by (str):
            demo (bool):
            host (str):
            id (str):
            kind (PushSiteKind):
            label (str):
            linked (int):
            origin (str):
            prompts (list[PushSitePrompt]):
            subscribers (int):
            worker (PushWorker):
            icon_url (str | Unset):
            snippet_seen_at (datetime.datetime | Unset):
     """

    added_at: datetime.datetime
    added_by: str
    demo: bool
    host: str
    id: str
    kind: PushSiteKind
    label: str
    linked: int
    origin: str
    prompts: list[PushSitePrompt]
    subscribers: int
    worker: PushWorker
    icon_url: str | Unset = UNSET
    snippet_seen_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_site_prompt import PushSitePrompt # noqa: PLC0415
        from ..models.push_worker import PushWorker # noqa: PLC0415
        added_at = self.added_at.isoformat()

        added_by = self.added_by

        demo = self.demo

        host = self.host

        id = self.id

        kind = self.kind.value

        label = self.label

        linked = self.linked

        origin = self.origin

        prompts = []
        for prompts_item_data in self.prompts:
            prompts_item = prompts_item_data.to_dict()
            prompts.append(prompts_item)



        subscribers = self.subscribers

        worker = self.worker.to_dict()

        icon_url = self.icon_url

        snippet_seen_at: str | Unset = UNSET
        if not isinstance(self.snippet_seen_at, Unset):
            snippet_seen_at = self.snippet_seen_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "addedAt": added_at,
            "addedBy": added_by,
            "demo": demo,
            "host": host,
            "id": id,
            "kind": kind,
            "label": label,
            "linked": linked,
            "origin": origin,
            "prompts": prompts,
            "subscribers": subscribers,
            "worker": worker,
        })
        if icon_url is not UNSET:
            field_dict["iconUrl"] = icon_url
        if snippet_seen_at is not UNSET:
            field_dict["snippetSeenAt"] = snippet_seen_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_site_prompt import PushSitePrompt # noqa: PLC0415
        from ..models.push_worker import PushWorker # noqa: PLC0415
        d = dict(src_dict)
        added_at = datetime.datetime.fromisoformat(d.pop("addedAt"))




        added_by = d.pop("addedBy")

        demo = d.pop("demo")

        host = d.pop("host")

        id = d.pop("id")

        kind = PushSiteKind(d.pop("kind"))




        label = d.pop("label")

        linked = d.pop("linked")

        origin = d.pop("origin")

        prompts = []
        _prompts = d.pop("prompts")
        for prompts_item_data in (_prompts):
            prompts_item = PushSitePrompt.from_dict(prompts_item_data)



            prompts.append(prompts_item)


        subscribers = d.pop("subscribers")

        worker = PushWorker.from_dict(d.pop("worker"))




        icon_url = d.pop("iconUrl", UNSET)

        _snippet_seen_at = d.pop("snippetSeenAt", UNSET)
        snippet_seen_at: datetime.datetime | Unset
        if isinstance(_snippet_seen_at,  Unset):
            snippet_seen_at = UNSET
        else:
            snippet_seen_at = datetime.datetime.fromisoformat(_snippet_seen_at)




        push_site = cls(
            added_at=added_at,
            added_by=added_by,
            demo=demo,
            host=host,
            id=id,
            kind=kind,
            label=label,
            linked=linked,
            origin=origin,
            prompts=prompts,
            subscribers=subscribers,
            worker=worker,
            icon_url=icon_url,
            snippet_seen_at=snippet_seen_at,
        )

        return push_site

