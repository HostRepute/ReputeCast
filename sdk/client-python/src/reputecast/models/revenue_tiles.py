from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="RevenueTiles")



@_attrs_define
class RevenueTiles:
    """ 
        Attributes:
            arr_cents (int):
            churn_base (int):
            churned (int):
            free_base (int):
            free_to_paid (int):
            gifts_cents (int):
            members (int):
            members_delta (int):
            mrr_cents (int):
            mrr_delta_cents (int):
            new (int):
            sponsors_cents (int):
            trialing (int):
            churn_rate (float | Unset):
            free_to_paid_rate (float | Unset):
            mrr_delta_pct (float | Unset):
     """

    arr_cents: int
    churn_base: int
    churned: int
    free_base: int
    free_to_paid: int
    gifts_cents: int
    members: int
    members_delta: int
    mrr_cents: int
    mrr_delta_cents: int
    new: int
    sponsors_cents: int
    trialing: int
    churn_rate: float | Unset = UNSET
    free_to_paid_rate: float | Unset = UNSET
    mrr_delta_pct: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        arr_cents = self.arr_cents

        churn_base = self.churn_base

        churned = self.churned

        free_base = self.free_base

        free_to_paid = self.free_to_paid

        gifts_cents = self.gifts_cents

        members = self.members

        members_delta = self.members_delta

        mrr_cents = self.mrr_cents

        mrr_delta_cents = self.mrr_delta_cents

        new = self.new

        sponsors_cents = self.sponsors_cents

        trialing = self.trialing

        churn_rate = self.churn_rate

        free_to_paid_rate = self.free_to_paid_rate

        mrr_delta_pct = self.mrr_delta_pct


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "arrCents": arr_cents,
            "churnBase": churn_base,
            "churned": churned,
            "freeBase": free_base,
            "freeToPaid": free_to_paid,
            "giftsCents": gifts_cents,
            "members": members,
            "membersDelta": members_delta,
            "mrrCents": mrr_cents,
            "mrrDeltaCents": mrr_delta_cents,
            "new": new,
            "sponsorsCents": sponsors_cents,
            "trialing": trialing,
        })
        if churn_rate is not UNSET:
            field_dict["churnRate"] = churn_rate
        if free_to_paid_rate is not UNSET:
            field_dict["freeToPaidRate"] = free_to_paid_rate
        if mrr_delta_pct is not UNSET:
            field_dict["mrrDeltaPct"] = mrr_delta_pct

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        arr_cents = d.pop("arrCents")

        churn_base = d.pop("churnBase")

        churned = d.pop("churned")

        free_base = d.pop("freeBase")

        free_to_paid = d.pop("freeToPaid")

        gifts_cents = d.pop("giftsCents")

        members = d.pop("members")

        members_delta = d.pop("membersDelta")

        mrr_cents = d.pop("mrrCents")

        mrr_delta_cents = d.pop("mrrDeltaCents")

        new = d.pop("new")

        sponsors_cents = d.pop("sponsorsCents")

        trialing = d.pop("trialing")

        churn_rate = d.pop("churnRate", UNSET)

        free_to_paid_rate = d.pop("freeToPaidRate", UNSET)

        mrr_delta_pct = d.pop("mrrDeltaPct", UNSET)

        revenue_tiles = cls(
            arr_cents=arr_cents,
            churn_base=churn_base,
            churned=churned,
            free_base=free_base,
            free_to_paid=free_to_paid,
            gifts_cents=gifts_cents,
            members=members,
            members_delta=members_delta,
            mrr_cents=mrr_cents,
            mrr_delta_cents=mrr_delta_cents,
            new=new,
            sponsors_cents=sponsors_cents,
            trialing=trialing,
            churn_rate=churn_rate,
            free_to_paid_rate=free_to_paid_rate,
            mrr_delta_pct=mrr_delta_pct,
        )

        return revenue_tiles

