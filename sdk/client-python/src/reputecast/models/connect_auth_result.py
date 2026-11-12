from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_auth_result_dkim import ConnectAuthResultDkim
from ..models.connect_auth_result_dmarc import ConnectAuthResultDmarc
from ..models.connect_auth_result_spf import ConnectAuthResultSpf






T = TypeVar("T", bound="ConnectAuthResult")



@_attrs_define
class ConnectAuthResult:
    """ 
        Attributes:
            clipped (bool):
            dkim (ConnectAuthResultDkim):
            dmarc (ConnectAuthResultDmarc):
            header (str):
            receiver (str):
            size_kb (int):
            spf (ConnectAuthResultSpf):
     """

    clipped: bool
    dkim: ConnectAuthResultDkim
    dmarc: ConnectAuthResultDmarc
    header: str
    receiver: str
    size_kb: int
    spf: ConnectAuthResultSpf





    def to_dict(self) -> dict[str, Any]:
        clipped = self.clipped

        dkim = self.dkim.value

        dmarc = self.dmarc.value

        header = self.header

        receiver = self.receiver

        size_kb = self.size_kb

        spf = self.spf.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clipped": clipped,
            "dkim": dkim,
            "dmarc": dmarc,
            "header": header,
            "receiver": receiver,
            "sizeKb": size_kb,
            "spf": spf,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clipped = d.pop("clipped")

        dkim = ConnectAuthResultDkim(d.pop("dkim"))




        dmarc = ConnectAuthResultDmarc(d.pop("dmarc"))




        header = d.pop("header")

        receiver = d.pop("receiver")

        size_kb = d.pop("sizeKb")

        spf = ConnectAuthResultSpf(d.pop("spf"))




        connect_auth_result = cls(
            clipped=clipped,
            dkim=dkim,
            dmarc=dmarc,
            header=header,
            receiver=receiver,
            size_kb=size_kb,
            spf=spf,
        )

        return connect_auth_result

