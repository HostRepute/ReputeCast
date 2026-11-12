from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.revenue_upgrade_row_kind import RevenueUpgradeRowKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="RevenueUpgradeRow")



@_attrs_define
class RevenueUpgradeRow:
    """ 
        Attributes:
            kind (RevenueUpgradeRowKind):
            members (int):
            mrr_cents (int):
            name (str):
            id (str | Unset):
            sent_at (datetime.datetime | Unset):
     """

    kind: RevenueUpgradeRowKind
    members: int
    mrr_cents: int
    name: str
    id: str | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        members = self.members

        mrr_cents = self.mrr_cents

        name = self.name

        id = self.id

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "members": members,
            "mrrCents": mrr_cents,
            "name": name,
        })
        if id is not UNSET:
            field_dict["id"] = id
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = RevenueUpgradeRowKind(d.pop("kind"))




        members = d.pop("members")

        mrr_cents = d.pop("mrrCents")

        name = d.pop("name")

        id = d.pop("id", UNSET)

        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        revenue_upgrade_row = cls(
            kind=kind,
            members=members,
            mrr_cents=mrr_cents,
            name=name,
            id=id,
            sent_at=sent_at,
        )

        return revenue_upgrade_row

