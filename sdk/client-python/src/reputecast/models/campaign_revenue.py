from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.campaign_conversions import CampaignConversions
  from ..models.campaign_payment import CampaignPayment
  from ..models.campaign_upgrade_line import CampaignUpgradeLine





T = TypeVar("T", bound="CampaignRevenue")



@_attrs_define
class CampaignRevenue:
    """ 
        Attributes:
            active (bool):
            checkouts (int):
            collected (list[CampaignPayment]):
            collected_cents (int):
            conversions (CampaignConversions):
            currency (str):
            delivered (int):
            has_upgrade_links (bool):
            helpers (list[str]):
            locked (bool):
            name (str):
            new_mrr_cents (int):
            paid_at_once (int):
            paid_lines (list[CampaignUpgradeLine]):
            pending (bool):
            potential_mrr_cents (int):
            trial_lines (list[CampaignUpgradeLine]):
            trials (int):
            upgrade_clicks (int):
            upgrades (int):
            window (int):
            sent_at (datetime.datetime | Unset):
            trials_end (datetime.datetime | Unset):
            upgraded_rate (float | Unset):
     """

    active: bool
    checkouts: int
    collected: list[CampaignPayment]
    collected_cents: int
    conversions: CampaignConversions
    currency: str
    delivered: int
    has_upgrade_links: bool
    helpers: list[str]
    locked: bool
    name: str
    new_mrr_cents: int
    paid_at_once: int
    paid_lines: list[CampaignUpgradeLine]
    pending: bool
    potential_mrr_cents: int
    trial_lines: list[CampaignUpgradeLine]
    trials: int
    upgrade_clicks: int
    upgrades: int
    window: int
    sent_at: datetime.datetime | Unset = UNSET
    trials_end: datetime.datetime | Unset = UNSET
    upgraded_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.campaign_conversions import CampaignConversions # noqa: PLC0415
        from ..models.campaign_payment import CampaignPayment # noqa: PLC0415
        from ..models.campaign_upgrade_line import CampaignUpgradeLine # noqa: PLC0415
        active = self.active

        checkouts = self.checkouts

        collected = []
        for collected_item_data in self.collected:
            collected_item = collected_item_data.to_dict()
            collected.append(collected_item)



        collected_cents = self.collected_cents

        conversions = self.conversions.to_dict()

        currency = self.currency

        delivered = self.delivered

        has_upgrade_links = self.has_upgrade_links

        helpers = self.helpers



        locked = self.locked

        name = self.name

        new_mrr_cents = self.new_mrr_cents

        paid_at_once = self.paid_at_once

        paid_lines = []
        for paid_lines_item_data in self.paid_lines:
            paid_lines_item = paid_lines_item_data.to_dict()
            paid_lines.append(paid_lines_item)



        pending = self.pending

        potential_mrr_cents = self.potential_mrr_cents

        trial_lines = []
        for trial_lines_item_data in self.trial_lines:
            trial_lines_item = trial_lines_item_data.to_dict()
            trial_lines.append(trial_lines_item)



        trials = self.trials

        upgrade_clicks = self.upgrade_clicks

        upgrades = self.upgrades

        window = self.window

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()

        trials_end: str | Unset = UNSET
        if not isinstance(self.trials_end, Unset):
            trials_end = self.trials_end.isoformat()

        upgraded_rate = self.upgraded_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "active": active,
            "checkouts": checkouts,
            "collected": collected,
            "collectedCents": collected_cents,
            "conversions": conversions,
            "currency": currency,
            "delivered": delivered,
            "hasUpgradeLinks": has_upgrade_links,
            "helpers": helpers,
            "locked": locked,
            "name": name,
            "newMrrCents": new_mrr_cents,
            "paidAtOnce": paid_at_once,
            "paidLines": paid_lines,
            "pending": pending,
            "potentialMrrCents": potential_mrr_cents,
            "trialLines": trial_lines,
            "trials": trials,
            "upgradeClicks": upgrade_clicks,
            "upgrades": upgrades,
            "window": window,
        })
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at
        if trials_end is not UNSET:
            field_dict["trialsEnd"] = trials_end
        if upgraded_rate is not UNSET:
            field_dict["upgradedRate"] = upgraded_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.campaign_conversions import CampaignConversions # noqa: PLC0415
        from ..models.campaign_payment import CampaignPayment # noqa: PLC0415
        from ..models.campaign_upgrade_line import CampaignUpgradeLine # noqa: PLC0415
        d = dict(src_dict)
        active = d.pop("active")

        checkouts = d.pop("checkouts")

        collected = []
        _collected = d.pop("collected")
        for collected_item_data in (_collected):
            collected_item = CampaignPayment.from_dict(collected_item_data)



            collected.append(collected_item)


        collected_cents = d.pop("collectedCents")

        conversions = CampaignConversions.from_dict(d.pop("conversions"))




        currency = d.pop("currency")

        delivered = d.pop("delivered")

        has_upgrade_links = d.pop("hasUpgradeLinks")

        helpers = cast(list[str], d.pop("helpers"))


        locked = d.pop("locked")

        name = d.pop("name")

        new_mrr_cents = d.pop("newMrrCents")

        paid_at_once = d.pop("paidAtOnce")

        paid_lines = []
        _paid_lines = d.pop("paidLines")
        for paid_lines_item_data in (_paid_lines):
            paid_lines_item = CampaignUpgradeLine.from_dict(paid_lines_item_data)



            paid_lines.append(paid_lines_item)


        pending = d.pop("pending")

        potential_mrr_cents = d.pop("potentialMrrCents")

        trial_lines = []
        _trial_lines = d.pop("trialLines")
        for trial_lines_item_data in (_trial_lines):
            trial_lines_item = CampaignUpgradeLine.from_dict(trial_lines_item_data)



            trial_lines.append(trial_lines_item)


        trials = d.pop("trials")

        upgrade_clicks = d.pop("upgradeClicks")

        upgrades = d.pop("upgrades")

        window = d.pop("window")

        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        _trials_end = d.pop("trialsEnd", UNSET)
        trials_end: datetime.datetime | Unset
        if isinstance(_trials_end,  Unset):
            trials_end = UNSET
        else:
            trials_end = datetime.datetime.fromisoformat(_trials_end)




        upgraded_rate = d.pop("upgradedRate", UNSET)

        campaign_revenue = cls(
            active=active,
            checkouts=checkouts,
            collected=collected,
            collected_cents=collected_cents,
            conversions=conversions,
            currency=currency,
            delivered=delivered,
            has_upgrade_links=has_upgrade_links,
            helpers=helpers,
            locked=locked,
            name=name,
            new_mrr_cents=new_mrr_cents,
            paid_at_once=paid_at_once,
            paid_lines=paid_lines,
            pending=pending,
            potential_mrr_cents=potential_mrr_cents,
            trial_lines=trial_lines,
            trials=trials,
            upgrade_clicks=upgrade_clicks,
            upgrades=upgrades,
            window=window,
            sent_at=sent_at,
            trials_end=trials_end,
            upgraded_rate=upgraded_rate,
        )

        return campaign_revenue

