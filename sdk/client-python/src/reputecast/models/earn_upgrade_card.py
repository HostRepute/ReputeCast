from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="EarnUpgradeCard")



@_attrs_define
class EarnUpgradeCard:
    """ 
        Attributes:
            checkouts_started (int):
            completed (int):
            host (str):
            live (bool):
            paid_at_once (int):
            test_mode (bool):
            trials (int):
            url (str):
            visits (int):
            test_checkout_url (str | Unset):
     """

    checkouts_started: int
    completed: int
    host: str
    live: bool
    paid_at_once: int
    test_mode: bool
    trials: int
    url: str
    visits: int
    test_checkout_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        checkouts_started = self.checkouts_started

        completed = self.completed

        host = self.host

        live = self.live

        paid_at_once = self.paid_at_once

        test_mode = self.test_mode

        trials = self.trials

        url = self.url

        visits = self.visits

        test_checkout_url = self.test_checkout_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checkoutsStarted": checkouts_started,
            "completed": completed,
            "host": host,
            "live": live,
            "paidAtOnce": paid_at_once,
            "testMode": test_mode,
            "trials": trials,
            "url": url,
            "visits": visits,
        })
        if test_checkout_url is not UNSET:
            field_dict["testCheckoutUrl"] = test_checkout_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        checkouts_started = d.pop("checkoutsStarted")

        completed = d.pop("completed")

        host = d.pop("host")

        live = d.pop("live")

        paid_at_once = d.pop("paidAtOnce")

        test_mode = d.pop("testMode")

        trials = d.pop("trials")

        url = d.pop("url")

        visits = d.pop("visits")

        test_checkout_url = d.pop("testCheckoutUrl", UNSET)

        earn_upgrade_card = cls(
            checkouts_started=checkouts_started,
            completed=completed,
            host=host,
            live=live,
            paid_at_once=paid_at_once,
            test_mode=test_mode,
            trials=trials,
            url=url,
            visits=visits,
            test_checkout_url=test_checkout_url,
        )

        return earn_upgrade_card

