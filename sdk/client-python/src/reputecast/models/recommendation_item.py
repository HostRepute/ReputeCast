from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.recommendation_item_invalid_reason import RecommendationItemInvalidReason
from ..models.recommendation_item_kind import RecommendationItemKind
from ..models.recommendation_item_state import RecommendationItemState
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.rec_image import RecImage





T = TypeVar("T", bound="RecommendationItem")



@_attrs_define
class RecommendationItem:
    """ 
        Attributes:
            clicks (int):
            confirmed (int):
            connected_by (str):
            daily_limit (int):
            description (str):
            hidden (bool):
            host (str):
            id (str):
            initials (str):
            invalid_reason (RecommendationItemInvalidReason):
            kind (RecommendationItemKind):
            link_id (str):
            name (str):
            paused_by (str):
            position (int):
            requests (int):
            show_block (bool):
            show_confirm (bool):
            shown (int):
            shown_confirm (int):
            shown_email (int):
            state (RecommendationItemState):
            today (int):
            url (str):
            waiting (int):
            connected_at (datetime.datetime | Unset):
            limit_until (datetime.datetime | Unset):
            logo (RecImage | Unset):
            since (datetime.datetime | Unset):
     """

    clicks: int
    confirmed: int
    connected_by: str
    daily_limit: int
    description: str
    hidden: bool
    host: str
    id: str
    initials: str
    invalid_reason: RecommendationItemInvalidReason
    kind: RecommendationItemKind
    link_id: str
    name: str
    paused_by: str
    position: int
    requests: int
    show_block: bool
    show_confirm: bool
    shown: int
    shown_confirm: int
    shown_email: int
    state: RecommendationItemState
    today: int
    url: str
    waiting: int
    connected_at: datetime.datetime | Unset = UNSET
    limit_until: datetime.datetime | Unset = UNSET
    logo: RecImage | Unset = UNSET
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.rec_image import RecImage # noqa: PLC0415
        clicks = self.clicks

        confirmed = self.confirmed

        connected_by = self.connected_by

        daily_limit = self.daily_limit

        description = self.description

        hidden = self.hidden

        host = self.host

        id = self.id

        initials = self.initials

        invalid_reason = self.invalid_reason.value

        kind = self.kind.value

        link_id = self.link_id

        name = self.name

        paused_by = self.paused_by

        position = self.position

        requests = self.requests

        show_block = self.show_block

        show_confirm = self.show_confirm

        shown = self.shown

        shown_confirm = self.shown_confirm

        shown_email = self.shown_email

        state = self.state.value

        today = self.today

        url = self.url

        waiting = self.waiting

        connected_at: str | Unset = UNSET
        if not isinstance(self.connected_at, Unset):
            connected_at = self.connected_at.isoformat()

        limit_until: str | Unset = UNSET
        if not isinstance(self.limit_until, Unset):
            limit_until = self.limit_until.isoformat()

        logo: dict[str, Any] | Unset = UNSET
        if not isinstance(self.logo, Unset):
            logo = self.logo.to_dict()

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clicks": clicks,
            "confirmed": confirmed,
            "connectedBy": connected_by,
            "dailyLimit": daily_limit,
            "description": description,
            "hidden": hidden,
            "host": host,
            "id": id,
            "initials": initials,
            "invalidReason": invalid_reason,
            "kind": kind,
            "linkId": link_id,
            "name": name,
            "pausedBy": paused_by,
            "position": position,
            "requests": requests,
            "showBlock": show_block,
            "showConfirm": show_confirm,
            "shown": shown,
            "shownConfirm": shown_confirm,
            "shownEmail": shown_email,
            "state": state,
            "today": today,
            "url": url,
            "waiting": waiting,
        })
        if connected_at is not UNSET:
            field_dict["connectedAt"] = connected_at
        if limit_until is not UNSET:
            field_dict["limitUntil"] = limit_until
        if logo is not UNSET:
            field_dict["logo"] = logo
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.rec_image import RecImage # noqa: PLC0415
        d = dict(src_dict)
        clicks = d.pop("clicks")

        confirmed = d.pop("confirmed")

        connected_by = d.pop("connectedBy")

        daily_limit = d.pop("dailyLimit")

        description = d.pop("description")

        hidden = d.pop("hidden")

        host = d.pop("host")

        id = d.pop("id")

        initials = d.pop("initials")

        invalid_reason = RecommendationItemInvalidReason(d.pop("invalidReason"))




        kind = RecommendationItemKind(d.pop("kind"))




        link_id = d.pop("linkId")

        name = d.pop("name")

        paused_by = d.pop("pausedBy")

        position = d.pop("position")

        requests = d.pop("requests")

        show_block = d.pop("showBlock")

        show_confirm = d.pop("showConfirm")

        shown = d.pop("shown")

        shown_confirm = d.pop("shownConfirm")

        shown_email = d.pop("shownEmail")

        state = RecommendationItemState(d.pop("state"))




        today = d.pop("today")

        url = d.pop("url")

        waiting = d.pop("waiting")

        _connected_at = d.pop("connectedAt", UNSET)
        connected_at: datetime.datetime | Unset
        if isinstance(_connected_at,  Unset):
            connected_at = UNSET
        else:
            connected_at = datetime.datetime.fromisoformat(_connected_at)




        _limit_until = d.pop("limitUntil", UNSET)
        limit_until: datetime.datetime | Unset
        if isinstance(_limit_until,  Unset):
            limit_until = UNSET
        else:
            limit_until = datetime.datetime.fromisoformat(_limit_until)




        _logo = d.pop("logo", UNSET)
        logo: RecImage | Unset
        if isinstance(_logo,  Unset):
            logo = UNSET
        else:
            logo = RecImage.from_dict(_logo)




        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        recommendation_item = cls(
            clicks=clicks,
            confirmed=confirmed,
            connected_by=connected_by,
            daily_limit=daily_limit,
            description=description,
            hidden=hidden,
            host=host,
            id=id,
            initials=initials,
            invalid_reason=invalid_reason,
            kind=kind,
            link_id=link_id,
            name=name,
            paused_by=paused_by,
            position=position,
            requests=requests,
            show_block=show_block,
            show_confirm=show_confirm,
            shown=shown,
            shown_confirm=shown_confirm,
            shown_email=shown_email,
            state=state,
            today=today,
            url=url,
            waiting=waiting,
            connected_at=connected_at,
            limit_until=limit_until,
            logo=logo,
            since=since,
        )

        return recommendation_item

