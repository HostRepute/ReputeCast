from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_signal_kind import ReferralSignalKind
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.referral_network import ReferralNetwork





T = TypeVar("T", bound="ReferralSignal")



@_attrs_define
class ReferralSignal:
    """ 
        Attributes:
            all_ (bool):
            count (int):
            first (str):
            ip (str):
            kind (ReferralSignalKind):
            last (str):
            minutes (int):
            network (ReferralNetwork):
            threshold (int):
            from_ (datetime.datetime | Unset):
            to (datetime.datetime | Unset):
     """

    all_: bool
    count: int
    first: str
    ip: str
    kind: ReferralSignalKind
    last: str
    minutes: int
    network: ReferralNetwork
    threshold: int
    from_: datetime.datetime | Unset = UNSET
    to: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_network import ReferralNetwork # noqa: PLC0415
        all_ = self.all_

        count = self.count

        first = self.first

        ip = self.ip

        kind = self.kind.value

        last = self.last

        minutes = self.minutes

        network = self.network.to_dict()

        threshold = self.threshold

        from_: str | Unset = UNSET
        if not isinstance(self.from_, Unset):
            from_ = self.from_.isoformat()

        to: str | Unset = UNSET
        if not isinstance(self.to, Unset):
            to = self.to.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "all": all_,
            "count": count,
            "first": first,
            "ip": ip,
            "kind": kind,
            "last": last,
            "minutes": minutes,
            "network": network,
            "threshold": threshold,
        })
        if from_ is not UNSET:
            field_dict["from"] = from_
        if to is not UNSET:
            field_dict["to"] = to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_network import ReferralNetwork # noqa: PLC0415
        d = dict(src_dict)
        all_ = d.pop("all")

        count = d.pop("count")

        first = d.pop("first")

        ip = d.pop("ip")

        kind = ReferralSignalKind(d.pop("kind"))




        last = d.pop("last")

        minutes = d.pop("minutes")

        network = ReferralNetwork.from_dict(d.pop("network"))




        threshold = d.pop("threshold")

        _from_ = d.pop("from", UNSET)
        from_: datetime.datetime | Unset
        if isinstance(_from_,  Unset):
            from_ = UNSET
        else:
            from_ = datetime.datetime.fromisoformat(_from_)




        _to = d.pop("to", UNSET)
        to: datetime.datetime | Unset
        if isinstance(_to,  Unset):
            to = UNSET
        else:
            to = datetime.datetime.fromisoformat(_to)




        referral_signal = cls(
            all_=all_,
            count=count,
            first=first,
            ip=ip,
            kind=kind,
            last=last,
            minutes=minutes,
            network=network,
            threshold=threshold,
            from_=from_,
            to=to,
        )

        return referral_signal

