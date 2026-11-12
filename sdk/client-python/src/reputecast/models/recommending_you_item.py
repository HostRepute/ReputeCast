from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.recommending_you_item_state import RecommendingYouItemState
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="RecommendingYouItem")



@_attrs_define
class RecommendingYouItem:
    """ 
        Attributes:
            confirmed (int):
            connected_by (str):
            daily_limit (int):
            instance (str):
            label (str):
            link_id (str):
            list_name (str):
            name (str):
            paused_by (str):
            requests (int):
            since (datetime.datetime):
            state (RecommendingYouItemState):
            today (int):
            paused_at (datetime.datetime | Unset):
     """

    confirmed: int
    connected_by: str
    daily_limit: int
    instance: str
    label: str
    link_id: str
    list_name: str
    name: str
    paused_by: str
    requests: int
    since: datetime.datetime
    state: RecommendingYouItemState
    today: int
    paused_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        confirmed = self.confirmed

        connected_by = self.connected_by

        daily_limit = self.daily_limit

        instance = self.instance

        label = self.label

        link_id = self.link_id

        list_name = self.list_name

        name = self.name

        paused_by = self.paused_by

        requests = self.requests

        since = self.since.isoformat()

        state = self.state.value

        today = self.today

        paused_at: str | Unset = UNSET
        if not isinstance(self.paused_at, Unset):
            paused_at = self.paused_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "confirmed": confirmed,
            "connectedBy": connected_by,
            "dailyLimit": daily_limit,
            "instance": instance,
            "label": label,
            "linkId": link_id,
            "listName": list_name,
            "name": name,
            "pausedBy": paused_by,
            "requests": requests,
            "since": since,
            "state": state,
            "today": today,
        })
        if paused_at is not UNSET:
            field_dict["pausedAt"] = paused_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        confirmed = d.pop("confirmed")

        connected_by = d.pop("connectedBy")

        daily_limit = d.pop("dailyLimit")

        instance = d.pop("instance")

        label = d.pop("label")

        link_id = d.pop("linkId")

        list_name = d.pop("listName")

        name = d.pop("name")

        paused_by = d.pop("pausedBy")

        requests = d.pop("requests")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        state = RecommendingYouItemState(d.pop("state"))




        today = d.pop("today")

        _paused_at = d.pop("pausedAt", UNSET)
        paused_at: datetime.datetime | Unset
        if isinstance(_paused_at,  Unset):
            paused_at = UNSET
        else:
            paused_at = datetime.datetime.fromisoformat(_paused_at)




        recommending_you_item = cls(
            confirmed=confirmed,
            connected_by=connected_by,
            daily_limit=daily_limit,
            instance=instance,
            label=label,
            link_id=link_id,
            list_name=list_name,
            name=name,
            paused_by=paused_by,
            requests=requests,
            since=since,
            state=state,
            today=today,
            paused_at=paused_at,
        )

        return recommending_you_item

