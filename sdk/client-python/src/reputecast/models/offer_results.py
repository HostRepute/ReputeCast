from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="OfferResults")



@_attrs_define
class OfferResults:
    """ 
        Attributes:
            accepted (int):
            codes_issued (int):
            codes_redeemed (int):
            mrr_cents (int):
            new_members (int):
            redemptions (int):
            shown (int):
            accept_rate (float | Unset):
     """

    accepted: int
    codes_issued: int
    codes_redeemed: int
    mrr_cents: int
    new_members: int
    redemptions: int
    shown: int
    accept_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        accepted = self.accepted

        codes_issued = self.codes_issued

        codes_redeemed = self.codes_redeemed

        mrr_cents = self.mrr_cents

        new_members = self.new_members

        redemptions = self.redemptions

        shown = self.shown

        accept_rate = self.accept_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accepted": accepted,
            "codesIssued": codes_issued,
            "codesRedeemed": codes_redeemed,
            "mrrCents": mrr_cents,
            "newMembers": new_members,
            "redemptions": redemptions,
            "shown": shown,
        })
        if accept_rate is not UNSET:
            field_dict["acceptRate"] = accept_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        accepted = d.pop("accepted")

        codes_issued = d.pop("codesIssued")

        codes_redeemed = d.pop("codesRedeemed")

        mrr_cents = d.pop("mrrCents")

        new_members = d.pop("newMembers")

        redemptions = d.pop("redemptions")

        shown = d.pop("shown")

        accept_rate = d.pop("acceptRate", UNSET)

        offer_results = cls(
            accepted=accepted,
            codes_issued=codes_issued,
            codes_redeemed=codes_redeemed,
            mrr_cents=mrr_cents,
            new_members=new_members,
            redemptions=redemptions,
            shown=shown,
            accept_rate=accept_rate,
        )

        return offer_results

