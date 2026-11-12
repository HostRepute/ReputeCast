from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.revenue_overview_period import RevenueOverviewPeriod
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.revenue_activity import RevenueActivity
  from ..models.revenue_cancellations import RevenueCancellations
  from ..models.revenue_chart import RevenueChart
  from ..models.revenue_conversions import RevenueConversions
  from ..models.revenue_free_to_paid import RevenueFreeToPaid
  from ..models.revenue_movement import RevenueMovement
  from ..models.revenue_need import RevenueNeed
  from ..models.revenue_tier_row import RevenueTierRow
  from ..models.revenue_tiles import RevenueTiles
  from ..models.revenue_trials import RevenueTrials
  from ..models.revenue_upgrades import RevenueUpgrades





T = TypeVar("T", bound="RevenueOverview")



@_attrs_define
class RevenueOverview:
    """ 
        Attributes:
            account (str):
            activity (list[RevenueActivity]):
            can_edit (bool):
            can_manage (bool):
            can_payments (bool):
            can_sponsors (bool):
            cancellations (RevenueCancellations):
            chart (RevenueChart):
            conversions (RevenueConversions):
            currency (str):
            failed (list[str]):
            free_to_paid (RevenueFreeToPaid):
            from_ (datetime.datetime):
            had_history (bool):
            helpers (list[str]):
            movement (RevenueMovement):
            needs (list[RevenueNeed]):
            period (RevenueOverviewPeriod):
            read_only (bool):
            tiers (list[RevenueTierRow]):
            tiles (RevenueTiles):
            to (datetime.datetime):
            too_early (bool):
            trials (RevenueTrials):
            upgrades (RevenueUpgrades):
            connected_at (datetime.datetime | Unset):
            data_since (datetime.datetime | Unset):
            reference (str | Unset):
            shown_from (datetime.datetime | Unset):
     """

    account: str
    activity: list[RevenueActivity]
    can_edit: bool
    can_manage: bool
    can_payments: bool
    can_sponsors: bool
    cancellations: RevenueCancellations
    chart: RevenueChart
    conversions: RevenueConversions
    currency: str
    failed: list[str]
    free_to_paid: RevenueFreeToPaid
    from_: datetime.datetime
    had_history: bool
    helpers: list[str]
    movement: RevenueMovement
    needs: list[RevenueNeed]
    period: RevenueOverviewPeriod
    read_only: bool
    tiers: list[RevenueTierRow]
    tiles: RevenueTiles
    to: datetime.datetime
    too_early: bool
    trials: RevenueTrials
    upgrades: RevenueUpgrades
    connected_at: datetime.datetime | Unset = UNSET
    data_since: datetime.datetime | Unset = UNSET
    reference: str | Unset = UNSET
    shown_from: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.revenue_activity import RevenueActivity # noqa: PLC0415
        from ..models.revenue_cancellations import RevenueCancellations # noqa: PLC0415
        from ..models.revenue_chart import RevenueChart # noqa: PLC0415
        from ..models.revenue_conversions import RevenueConversions # noqa: PLC0415
        from ..models.revenue_free_to_paid import RevenueFreeToPaid # noqa: PLC0415
        from ..models.revenue_movement import RevenueMovement # noqa: PLC0415
        from ..models.revenue_need import RevenueNeed # noqa: PLC0415
        from ..models.revenue_tier_row import RevenueTierRow # noqa: PLC0415
        from ..models.revenue_tiles import RevenueTiles # noqa: PLC0415
        from ..models.revenue_trials import RevenueTrials # noqa: PLC0415
        from ..models.revenue_upgrades import RevenueUpgrades # noqa: PLC0415
        account = self.account

        activity = []
        for activity_item_data in self.activity:
            activity_item = activity_item_data.to_dict()
            activity.append(activity_item)



        can_edit = self.can_edit

        can_manage = self.can_manage

        can_payments = self.can_payments

        can_sponsors = self.can_sponsors

        cancellations = self.cancellations.to_dict()

        chart = self.chart.to_dict()

        conversions = self.conversions.to_dict()

        currency = self.currency

        failed = self.failed



        free_to_paid = self.free_to_paid.to_dict()

        from_ = self.from_.isoformat()

        had_history = self.had_history

        helpers = self.helpers



        movement = self.movement.to_dict()

        needs = []
        for needs_item_data in self.needs:
            needs_item = needs_item_data.to_dict()
            needs.append(needs_item)



        period = self.period.value

        read_only = self.read_only

        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)



        tiles = self.tiles.to_dict()

        to = self.to.isoformat()

        too_early = self.too_early

        trials = self.trials.to_dict()

        upgrades = self.upgrades.to_dict()

        connected_at: str | Unset = UNSET
        if not isinstance(self.connected_at, Unset):
            connected_at = self.connected_at.isoformat()

        data_since: str | Unset = UNSET
        if not isinstance(self.data_since, Unset):
            data_since = self.data_since.isoformat()

        reference = self.reference

        shown_from: str | Unset = UNSET
        if not isinstance(self.shown_from, Unset):
            shown_from = self.shown_from.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "account": account,
            "activity": activity,
            "canEdit": can_edit,
            "canManage": can_manage,
            "canPayments": can_payments,
            "canSponsors": can_sponsors,
            "cancellations": cancellations,
            "chart": chart,
            "conversions": conversions,
            "currency": currency,
            "failed": failed,
            "freeToPaid": free_to_paid,
            "from": from_,
            "hadHistory": had_history,
            "helpers": helpers,
            "movement": movement,
            "needs": needs,
            "period": period,
            "readOnly": read_only,
            "tiers": tiers,
            "tiles": tiles,
            "to": to,
            "tooEarly": too_early,
            "trials": trials,
            "upgrades": upgrades,
        })
        if connected_at is not UNSET:
            field_dict["connectedAt"] = connected_at
        if data_since is not UNSET:
            field_dict["dataSince"] = data_since
        if reference is not UNSET:
            field_dict["reference"] = reference
        if shown_from is not UNSET:
            field_dict["shownFrom"] = shown_from

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.revenue_activity import RevenueActivity # noqa: PLC0415
        from ..models.revenue_cancellations import RevenueCancellations # noqa: PLC0415
        from ..models.revenue_chart import RevenueChart # noqa: PLC0415
        from ..models.revenue_conversions import RevenueConversions # noqa: PLC0415
        from ..models.revenue_free_to_paid import RevenueFreeToPaid # noqa: PLC0415
        from ..models.revenue_movement import RevenueMovement # noqa: PLC0415
        from ..models.revenue_need import RevenueNeed # noqa: PLC0415
        from ..models.revenue_tier_row import RevenueTierRow # noqa: PLC0415
        from ..models.revenue_tiles import RevenueTiles # noqa: PLC0415
        from ..models.revenue_trials import RevenueTrials # noqa: PLC0415
        from ..models.revenue_upgrades import RevenueUpgrades # noqa: PLC0415
        d = dict(src_dict)
        account = d.pop("account")

        activity = []
        _activity = d.pop("activity")
        for activity_item_data in (_activity):
            activity_item = RevenueActivity.from_dict(activity_item_data)



            activity.append(activity_item)


        can_edit = d.pop("canEdit")

        can_manage = d.pop("canManage")

        can_payments = d.pop("canPayments")

        can_sponsors = d.pop("canSponsors")

        cancellations = RevenueCancellations.from_dict(d.pop("cancellations"))




        chart = RevenueChart.from_dict(d.pop("chart"))




        conversions = RevenueConversions.from_dict(d.pop("conversions"))




        currency = d.pop("currency")

        failed = cast(list[str], d.pop("failed"))


        free_to_paid = RevenueFreeToPaid.from_dict(d.pop("freeToPaid"))




        from_ = datetime.datetime.fromisoformat(d.pop("from"))




        had_history = d.pop("hadHistory")

        helpers = cast(list[str], d.pop("helpers"))


        movement = RevenueMovement.from_dict(d.pop("movement"))




        needs = []
        _needs = d.pop("needs")
        for needs_item_data in (_needs):
            needs_item = RevenueNeed.from_dict(needs_item_data)



            needs.append(needs_item)


        period = RevenueOverviewPeriod(d.pop("period"))




        read_only = d.pop("readOnly")

        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = RevenueTierRow.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        tiles = RevenueTiles.from_dict(d.pop("tiles"))




        to = datetime.datetime.fromisoformat(d.pop("to"))




        too_early = d.pop("tooEarly")

        trials = RevenueTrials.from_dict(d.pop("trials"))




        upgrades = RevenueUpgrades.from_dict(d.pop("upgrades"))




        _connected_at = d.pop("connectedAt", UNSET)
        connected_at: datetime.datetime | Unset
        if isinstance(_connected_at,  Unset):
            connected_at = UNSET
        else:
            connected_at = datetime.datetime.fromisoformat(_connected_at)




        _data_since = d.pop("dataSince", UNSET)
        data_since: datetime.datetime | Unset
        if isinstance(_data_since,  Unset):
            data_since = UNSET
        else:
            data_since = datetime.datetime.fromisoformat(_data_since)




        reference = d.pop("reference", UNSET)

        _shown_from = d.pop("shownFrom", UNSET)
        shown_from: datetime.datetime | Unset
        if isinstance(_shown_from,  Unset):
            shown_from = UNSET
        else:
            shown_from = datetime.datetime.fromisoformat(_shown_from)




        revenue_overview = cls(
            account=account,
            activity=activity,
            can_edit=can_edit,
            can_manage=can_manage,
            can_payments=can_payments,
            can_sponsors=can_sponsors,
            cancellations=cancellations,
            chart=chart,
            conversions=conversions,
            currency=currency,
            failed=failed,
            free_to_paid=free_to_paid,
            from_=from_,
            had_history=had_history,
            helpers=helpers,
            movement=movement,
            needs=needs,
            period=period,
            read_only=read_only,
            tiers=tiers,
            tiles=tiles,
            to=to,
            too_early=too_early,
            trials=trials,
            upgrades=upgrades,
            connected_at=connected_at,
            data_since=data_since,
            reference=reference,
            shown_from=shown_from,
        )

        return revenue_overview

