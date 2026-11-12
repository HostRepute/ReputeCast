from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.scoring_use_kind import ScoringUseKind
from ..models.scoring_use_show import ScoringUseShow
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ScoringUse")



@_attrs_define
class ScoringUse:
    """ 
        Attributes:
            entered (int):
            id (str):
            kind (ScoringUseKind):
            name (str):
            by (str | Unset):
            conditions (Any | Unset):
            contacts (int | Unset):
            direction (str | Unset):
            href (str | Unset):
            of (int | Unset):
            once_in_days (int | Unset):
            show (ScoringUseShow | Unset):
            since (datetime.datetime | Unset):
            status (str | Unset):
            threshold (int | Unset):
            where (str | Unset):
     """

    entered: int
    id: str
    kind: ScoringUseKind
    name: str
    by: str | Unset = UNSET
    conditions: Any | Unset = UNSET
    contacts: int | Unset = UNSET
    direction: str | Unset = UNSET
    href: str | Unset = UNSET
    of: int | Unset = UNSET
    once_in_days: int | Unset = UNSET
    show: ScoringUseShow | Unset = UNSET
    since: datetime.datetime | Unset = UNSET
    status: str | Unset = UNSET
    threshold: int | Unset = UNSET
    where: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        entered = self.entered

        id = self.id

        kind = self.kind.value

        name = self.name

        by = self.by

        conditions = self.conditions

        contacts = self.contacts

        direction = self.direction

        href = self.href

        of = self.of

        once_in_days = self.once_in_days

        show: str | Unset = UNSET
        if not isinstance(self.show, Unset):
            show = self.show.value


        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()

        status = self.status

        threshold = self.threshold

        where = self.where


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "entered": entered,
            "id": id,
            "kind": kind,
            "name": name,
        })
        if by is not UNSET:
            field_dict["by"] = by
        if conditions is not UNSET:
            field_dict["conditions"] = conditions
        if contacts is not UNSET:
            field_dict["contacts"] = contacts
        if direction is not UNSET:
            field_dict["direction"] = direction
        if href is not UNSET:
            field_dict["href"] = href
        if of is not UNSET:
            field_dict["of"] = of
        if once_in_days is not UNSET:
            field_dict["onceInDays"] = once_in_days
        if show is not UNSET:
            field_dict["show"] = show
        if since is not UNSET:
            field_dict["since"] = since
        if status is not UNSET:
            field_dict["status"] = status
        if threshold is not UNSET:
            field_dict["threshold"] = threshold
        if where is not UNSET:
            field_dict["where"] = where

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        entered = d.pop("entered")

        id = d.pop("id")

        kind = ScoringUseKind(d.pop("kind"))




        name = d.pop("name")

        by = d.pop("by", UNSET)

        conditions = d.pop("conditions", UNSET)

        contacts = d.pop("contacts", UNSET)

        direction = d.pop("direction", UNSET)

        href = d.pop("href", UNSET)

        of = d.pop("of", UNSET)

        once_in_days = d.pop("onceInDays", UNSET)

        _show = d.pop("show", UNSET)
        show: ScoringUseShow | Unset
        if isinstance(_show,  Unset):
            show = UNSET
        else:
            show = ScoringUseShow(_show)




        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        status = d.pop("status", UNSET)

        threshold = d.pop("threshold", UNSET)

        where = d.pop("where", UNSET)

        scoring_use = cls(
            entered=entered,
            id=id,
            kind=kind,
            name=name,
            by=by,
            conditions=conditions,
            contacts=contacts,
            direction=direction,
            href=href,
            of=of,
            once_in_days=once_in_days,
            show=show,
            since=since,
            status=status,
            threshold=threshold,
            where=where,
        )

        return scoring_use

