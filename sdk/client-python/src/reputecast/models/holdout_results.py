from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.holdout_results_verdict import HoldoutResultsVerdict
from ..models.holdout_results_window import HoldoutResultsWindow
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.holdout_group_result import HoldoutGroupResult
  from ..models.holdout_metric import HoldoutMetric
  from ..models.holdout_other_currency import HoldoutOtherCurrency
  from ..models.holdout_period_item import HoldoutPeriodItem





T = TypeVar("T", bound="HoldoutResults")



@_attrs_define
class HoldoutResults:
    """ 
        Attributes:
            campaigns_included (int):
            computing (bool):
            currency (str):
            holdout (HoldoutGroupResult):
            marketing (HoldoutGroupResult):
            metrics (list[HoldoutMetric]):
            min_holdout (int):
            other_currencies (list[HoldoutOtherCurrency]):
            periods (list[HoldoutPeriodItem]):
            tracking (bool):
            verdict (HoldoutResultsVerdict):
            window (HoldoutResultsWindow):
            computed_at (datetime.datetime | Unset):
            from_ (datetime.datetime | Unset):
            next_measure_at (datetime.datetime | Unset):
            period (HoldoutPeriodItem | Unset):
            to (datetime.datetime | Unset):
     """

    campaigns_included: int
    computing: bool
    currency: str
    holdout: HoldoutGroupResult
    marketing: HoldoutGroupResult
    metrics: list[HoldoutMetric]
    min_holdout: int
    other_currencies: list[HoldoutOtherCurrency]
    periods: list[HoldoutPeriodItem]
    tracking: bool
    verdict: HoldoutResultsVerdict
    window: HoldoutResultsWindow
    computed_at: datetime.datetime | Unset = UNSET
    from_: datetime.datetime | Unset = UNSET
    next_measure_at: datetime.datetime | Unset = UNSET
    period: HoldoutPeriodItem | Unset = UNSET
    to: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.holdout_group_result import HoldoutGroupResult # noqa: PLC0415
        from ..models.holdout_metric import HoldoutMetric # noqa: PLC0415
        from ..models.holdout_other_currency import HoldoutOtherCurrency # noqa: PLC0415
        from ..models.holdout_period_item import HoldoutPeriodItem # noqa: PLC0415
        campaigns_included = self.campaigns_included

        computing = self.computing

        currency = self.currency

        holdout = self.holdout.to_dict()

        marketing = self.marketing.to_dict()

        metrics = []
        for metrics_item_data in self.metrics:
            metrics_item = metrics_item_data.to_dict()
            metrics.append(metrics_item)



        min_holdout = self.min_holdout

        other_currencies = []
        for other_currencies_item_data in self.other_currencies:
            other_currencies_item = other_currencies_item_data.to_dict()
            other_currencies.append(other_currencies_item)



        periods = []
        for periods_item_data in self.periods:
            periods_item = periods_item_data.to_dict()
            periods.append(periods_item)



        tracking = self.tracking

        verdict = self.verdict.value

        window = self.window.value

        computed_at: str | Unset = UNSET
        if not isinstance(self.computed_at, Unset):
            computed_at = self.computed_at.isoformat()

        from_: str | Unset = UNSET
        if not isinstance(self.from_, Unset):
            from_ = self.from_.isoformat()

        next_measure_at: str | Unset = UNSET
        if not isinstance(self.next_measure_at, Unset):
            next_measure_at = self.next_measure_at.isoformat()

        period: dict[str, Any] | Unset = UNSET
        if not isinstance(self.period, Unset):
            period = self.period.to_dict()

        to: str | Unset = UNSET
        if not isinstance(self.to, Unset):
            to = self.to.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaignsIncluded": campaigns_included,
            "computing": computing,
            "currency": currency,
            "holdout": holdout,
            "marketing": marketing,
            "metrics": metrics,
            "minHoldout": min_holdout,
            "otherCurrencies": other_currencies,
            "periods": periods,
            "tracking": tracking,
            "verdict": verdict,
            "window": window,
        })
        if computed_at is not UNSET:
            field_dict["computedAt"] = computed_at
        if from_ is not UNSET:
            field_dict["from"] = from_
        if next_measure_at is not UNSET:
            field_dict["nextMeasureAt"] = next_measure_at
        if period is not UNSET:
            field_dict["period"] = period
        if to is not UNSET:
            field_dict["to"] = to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.holdout_group_result import HoldoutGroupResult # noqa: PLC0415
        from ..models.holdout_metric import HoldoutMetric # noqa: PLC0415
        from ..models.holdout_other_currency import HoldoutOtherCurrency # noqa: PLC0415
        from ..models.holdout_period_item import HoldoutPeriodItem # noqa: PLC0415
        d = dict(src_dict)
        campaigns_included = d.pop("campaignsIncluded")

        computing = d.pop("computing")

        currency = d.pop("currency")

        holdout = HoldoutGroupResult.from_dict(d.pop("holdout"))




        marketing = HoldoutGroupResult.from_dict(d.pop("marketing"))




        metrics = []
        _metrics = d.pop("metrics")
        for metrics_item_data in (_metrics):
            metrics_item = HoldoutMetric.from_dict(metrics_item_data)



            metrics.append(metrics_item)


        min_holdout = d.pop("minHoldout")

        other_currencies = []
        _other_currencies = d.pop("otherCurrencies")
        for other_currencies_item_data in (_other_currencies):
            other_currencies_item = HoldoutOtherCurrency.from_dict(other_currencies_item_data)



            other_currencies.append(other_currencies_item)


        periods = []
        _periods = d.pop("periods")
        for periods_item_data in (_periods):
            periods_item = HoldoutPeriodItem.from_dict(periods_item_data)



            periods.append(periods_item)


        tracking = d.pop("tracking")

        verdict = HoldoutResultsVerdict(d.pop("verdict"))




        window = HoldoutResultsWindow(d.pop("window"))




        _computed_at = d.pop("computedAt", UNSET)
        computed_at: datetime.datetime | Unset
        if isinstance(_computed_at,  Unset):
            computed_at = UNSET
        else:
            computed_at = datetime.datetime.fromisoformat(_computed_at)




        _from_ = d.pop("from", UNSET)
        from_: datetime.datetime | Unset
        if isinstance(_from_,  Unset):
            from_ = UNSET
        else:
            from_ = datetime.datetime.fromisoformat(_from_)




        _next_measure_at = d.pop("nextMeasureAt", UNSET)
        next_measure_at: datetime.datetime | Unset
        if isinstance(_next_measure_at,  Unset):
            next_measure_at = UNSET
        else:
            next_measure_at = datetime.datetime.fromisoformat(_next_measure_at)




        _period = d.pop("period", UNSET)
        period: HoldoutPeriodItem | Unset
        if isinstance(_period,  Unset):
            period = UNSET
        else:
            period = HoldoutPeriodItem.from_dict(_period)




        _to = d.pop("to", UNSET)
        to: datetime.datetime | Unset
        if isinstance(_to,  Unset):
            to = UNSET
        else:
            to = datetime.datetime.fromisoformat(_to)




        holdout_results = cls(
            campaigns_included=campaigns_included,
            computing=computing,
            currency=currency,
            holdout=holdout,
            marketing=marketing,
            metrics=metrics,
            min_holdout=min_holdout,
            other_currencies=other_currencies,
            periods=periods,
            tracking=tracking,
            verdict=verdict,
            window=window,
            computed_at=computed_at,
            from_=from_,
            next_measure_at=next_measure_at,
            period=period,
            to=to,
        )

        return holdout_results

