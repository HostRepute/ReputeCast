from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_import_decisions_no_subscription import EarnImportDecisionsNoSubscription
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.earn_import_decisions_conflicts import EarnImportDecisionsConflicts
  from ..models.earn_import_decisions_prices import EarnImportDecisionsPrices





T = TypeVar("T", bound="EarnImportDecisions")



@_attrs_define
class EarnImportDecisions:
    """ 
        Attributes:
            no_subscription (EarnImportDecisionsNoSubscription):  Default: EarnImportDecisionsNoSubscription.COMPLIMENTARY.
            send_welcome (bool):
            conflicts (EarnImportDecisionsConflicts | Unset):
            prices (EarnImportDecisionsPrices | Unset):
     """

    send_welcome: bool
    no_subscription: EarnImportDecisionsNoSubscription = EarnImportDecisionsNoSubscription.COMPLIMENTARY
    conflicts: EarnImportDecisionsConflicts | Unset = UNSET
    prices: EarnImportDecisionsPrices | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_import_decisions_conflicts import EarnImportDecisionsConflicts # noqa: PLC0415
        from ..models.earn_import_decisions_prices import EarnImportDecisionsPrices # noqa: PLC0415
        no_subscription = self.no_subscription.value

        send_welcome = self.send_welcome

        conflicts: dict[str, Any] | Unset = UNSET
        if not isinstance(self.conflicts, Unset):
            conflicts = self.conflicts.to_dict()

        prices: dict[str, Any] | Unset = UNSET
        if not isinstance(self.prices, Unset):
            prices = self.prices.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "noSubscription": no_subscription,
            "sendWelcome": send_welcome,
        })
        if conflicts is not UNSET:
            field_dict["conflicts"] = conflicts
        if prices is not UNSET:
            field_dict["prices"] = prices

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_import_decisions_conflicts import EarnImportDecisionsConflicts # noqa: PLC0415
        from ..models.earn_import_decisions_prices import EarnImportDecisionsPrices # noqa: PLC0415
        d = dict(src_dict)
        no_subscription = EarnImportDecisionsNoSubscription(d.pop("noSubscription"))




        send_welcome = d.pop("sendWelcome")

        _conflicts = d.pop("conflicts", UNSET)
        conflicts: EarnImportDecisionsConflicts | Unset
        if isinstance(_conflicts,  Unset):
            conflicts = UNSET
        else:
            conflicts = EarnImportDecisionsConflicts.from_dict(_conflicts)




        _prices = d.pop("prices", UNSET)
        prices: EarnImportDecisionsPrices | Unset
        if isinstance(_prices,  Unset):
            prices = UNSET
        else:
            prices = EarnImportDecisionsPrices.from_dict(_prices)




        earn_import_decisions = cls(
            no_subscription=no_subscription,
            send_welcome=send_welcome,
            conflicts=conflicts,
            prices=prices,
        )

        return earn_import_decisions

