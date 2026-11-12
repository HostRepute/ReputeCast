from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_leaderboard_settings_periods_item import ReferralLeaderboardSettingsPeriodsItem
from ..models.referral_leaderboard_settings_rows import ReferralLeaderboardSettingsRows
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReferralLeaderboardSettings")



@_attrs_define
class ReferralLeaderboardSettings:
    """ 
        Attributes:
            by (str):
            on (bool):
            periods (list[ReferralLeaderboardSettingsPeriodsItem]):
            rows (ReferralLeaderboardSettingsRows):
            url (str):
            at (datetime.datetime | Unset):
     """

    by: str
    on: bool
    periods: list[ReferralLeaderboardSettingsPeriodsItem]
    rows: ReferralLeaderboardSettingsRows
    url: str
    at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        by = self.by

        on = self.on

        periods = []
        for periods_item_data in self.periods:
            periods_item = periods_item_data.value
            periods.append(periods_item)



        rows = self.rows.value

        url = self.url

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "by": by,
            "on": on,
            "periods": periods,
            "rows": rows,
            "url": url,
        })
        if at is not UNSET:
            field_dict["at"] = at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        by = d.pop("by")

        on = d.pop("on")

        periods = []
        _periods = d.pop("periods")
        for periods_item_data in (_periods):
            periods_item = ReferralLeaderboardSettingsPeriodsItem(periods_item_data)



            periods.append(periods_item)


        rows = ReferralLeaderboardSettingsRows(d.pop("rows"))




        url = d.pop("url")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        referral_leaderboard_settings = cls(
            by=by,
            on=on,
            periods=periods,
            rows=rows,
            url=url,
            at=at,
        )

        return referral_leaderboard_settings

