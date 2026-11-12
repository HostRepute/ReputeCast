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
  from ..models.check_line import CheckLine





T = TypeVar("T", bound="IntegrationReachability")



@_attrs_define
class IntegrationReachability:
    """ 
        Attributes:
            admins (list[str]):
            dns (CheckLine):
            private (bool):
            private_webhooks (bool):
            reach (CheckLine):
            reachable (bool):
            tls (CheckLine):
            url (str):
            checked_at (datetime.datetime | Unset):
     """

    admins: list[str]
    dns: CheckLine
    private: bool
    private_webhooks: bool
    reach: CheckLine
    reachable: bool
    tls: CheckLine
    url: str
    checked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.check_line import CheckLine # noqa: PLC0415
        admins = self.admins



        dns = self.dns.to_dict()

        private = self.private

        private_webhooks = self.private_webhooks

        reach = self.reach.to_dict()

        reachable = self.reachable

        tls = self.tls.to_dict()

        url = self.url

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "admins": admins,
            "dns": dns,
            "private": private,
            "privateWebhooks": private_webhooks,
            "reach": reach,
            "reachable": reachable,
            "tls": tls,
            "url": url,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.check_line import CheckLine # noqa: PLC0415
        d = dict(src_dict)
        admins = cast(list[str], d.pop("admins"))


        dns = CheckLine.from_dict(d.pop("dns"))




        private = d.pop("private")

        private_webhooks = d.pop("privateWebhooks")

        reach = CheckLine.from_dict(d.pop("reach"))




        reachable = d.pop("reachable")

        tls = CheckLine.from_dict(d.pop("tls"))




        url = d.pop("url")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        integration_reachability = cls(
            admins=admins,
            dns=dns,
            private=private,
            private_webhooks=private_webhooks,
            reach=reach,
            reachable=reachable,
            tls=tls,
            url=url,
            checked_at=checked_at,
        )

        return integration_reachability

