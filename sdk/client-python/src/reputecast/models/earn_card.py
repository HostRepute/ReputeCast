from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_card_state import EarnCardState
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.card_issue import CardIssue





T = TypeVar("T", bound="EarnCard")



@_attrs_define
class EarnCard:
    """ 
        Attributes:
            active (int):
            can_set_up (bool):
            currency (str):
            members (int):
            members_delta (int):
            mrr_cents (int):
            mrr_delta_cents (int):
            state (EarnCardState):
            trialing (int):
            account (str | Unset):
            churn_rate (float | Unset):
            mode (str | Unset):
            next_sponsor (CardIssue | Unset):
            provider_name (str | Unset):
            silent_hours (int | Unset):
     """

    active: int
    can_set_up: bool
    currency: str
    members: int
    members_delta: int
    mrr_cents: int
    mrr_delta_cents: int
    state: EarnCardState
    trialing: int
    account: str | Unset = UNSET
    churn_rate: float | Unset = UNSET
    mode: str | Unset = UNSET
    next_sponsor: CardIssue | Unset = UNSET
    provider_name: str | Unset = UNSET
    silent_hours: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.card_issue import CardIssue # noqa: PLC0415
        active = self.active

        can_set_up = self.can_set_up

        currency = self.currency

        members = self.members

        members_delta = self.members_delta

        mrr_cents = self.mrr_cents

        mrr_delta_cents = self.mrr_delta_cents

        state = self.state.value

        trialing = self.trialing

        account = self.account

        churn_rate = self.churn_rate

        mode = self.mode

        next_sponsor: dict[str, Any] | Unset = UNSET
        if not isinstance(self.next_sponsor, Unset):
            next_sponsor = self.next_sponsor.to_dict()

        provider_name = self.provider_name

        silent_hours = self.silent_hours


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "active": active,
            "canSetUp": can_set_up,
            "currency": currency,
            "members": members,
            "membersDelta": members_delta,
            "mrrCents": mrr_cents,
            "mrrDeltaCents": mrr_delta_cents,
            "state": state,
            "trialing": trialing,
        })
        if account is not UNSET:
            field_dict["account"] = account
        if churn_rate is not UNSET:
            field_dict["churnRate"] = churn_rate
        if mode is not UNSET:
            field_dict["mode"] = mode
        if next_sponsor is not UNSET:
            field_dict["nextSponsor"] = next_sponsor
        if provider_name is not UNSET:
            field_dict["providerName"] = provider_name
        if silent_hours is not UNSET:
            field_dict["silentHours"] = silent_hours

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.card_issue import CardIssue # noqa: PLC0415
        d = dict(src_dict)
        active = d.pop("active")

        can_set_up = d.pop("canSetUp")

        currency = d.pop("currency")

        members = d.pop("members")

        members_delta = d.pop("membersDelta")

        mrr_cents = d.pop("mrrCents")

        mrr_delta_cents = d.pop("mrrDeltaCents")

        state = EarnCardState(d.pop("state"))




        trialing = d.pop("trialing")

        account = d.pop("account", UNSET)

        churn_rate = d.pop("churnRate", UNSET)

        mode = d.pop("mode", UNSET)

        _next_sponsor = d.pop("nextSponsor", UNSET)
        next_sponsor: CardIssue | Unset
        if isinstance(_next_sponsor,  Unset):
            next_sponsor = UNSET
        else:
            next_sponsor = CardIssue.from_dict(_next_sponsor)




        provider_name = d.pop("providerName", UNSET)

        silent_hours = d.pop("silentHours", UNSET)

        earn_card = cls(
            active=active,
            can_set_up=can_set_up,
            currency=currency,
            members=members,
            members_delta=members_delta,
            mrr_cents=mrr_cents,
            mrr_delta_cents=mrr_delta_cents,
            state=state,
            trialing=trialing,
            account=account,
            churn_rate=churn_rate,
            mode=mode,
            next_sponsor=next_sponsor,
            provider_name=provider_name,
            silent_hours=silent_hours,
        )

        return earn_card

