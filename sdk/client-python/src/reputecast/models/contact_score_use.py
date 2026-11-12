from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_score_use_kind import ContactScoreUseKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ContactScoreUse")



@_attrs_define
class ContactScoreUse:
    """ 
        Attributes:
            id (str):
            in_ (bool):
            kind (ContactScoreUseKind):
            name (str):
            crossed_at (datetime.datetime | Unset):
            direction (str | Unset):
            href (str | Unset):
            live_since (datetime.datetime | Unset):
            threshold (int | Unset):
     """

    id: str
    in_: bool
    kind: ContactScoreUseKind
    name: str
    crossed_at: datetime.datetime | Unset = UNSET
    direction: str | Unset = UNSET
    href: str | Unset = UNSET
    live_since: datetime.datetime | Unset = UNSET
    threshold: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        in_ = self.in_

        kind = self.kind.value

        name = self.name

        crossed_at: str | Unset = UNSET
        if not isinstance(self.crossed_at, Unset):
            crossed_at = self.crossed_at.isoformat()

        direction = self.direction

        href = self.href

        live_since: str | Unset = UNSET
        if not isinstance(self.live_since, Unset):
            live_since = self.live_since.isoformat()

        threshold = self.threshold


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "in": in_,
            "kind": kind,
            "name": name,
        })
        if crossed_at is not UNSET:
            field_dict["crossedAt"] = crossed_at
        if direction is not UNSET:
            field_dict["direction"] = direction
        if href is not UNSET:
            field_dict["href"] = href
        if live_since is not UNSET:
            field_dict["liveSince"] = live_since
        if threshold is not UNSET:
            field_dict["threshold"] = threshold

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        in_ = d.pop("in")

        kind = ContactScoreUseKind(d.pop("kind"))




        name = d.pop("name")

        _crossed_at = d.pop("crossedAt", UNSET)
        crossed_at: datetime.datetime | Unset
        if isinstance(_crossed_at,  Unset):
            crossed_at = UNSET
        else:
            crossed_at = datetime.datetime.fromisoformat(_crossed_at)




        direction = d.pop("direction", UNSET)

        href = d.pop("href", UNSET)

        _live_since = d.pop("liveSince", UNSET)
        live_since: datetime.datetime | Unset
        if isinstance(_live_since,  Unset):
            live_since = UNSET
        else:
            live_since = datetime.datetime.fromisoformat(_live_since)




        threshold = d.pop("threshold", UNSET)

        contact_score_use = cls(
            id=id,
            in_=in_,
            kind=kind,
            name=name,
            crossed_at=crossed_at,
            direction=direction,
            href=href,
            live_since=live_since,
            threshold=threshold,
        )

        return contact_score_use

